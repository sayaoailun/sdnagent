// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package server

import (
	"context"
	"fmt"
	"strings"

	"github.com/digitalocean/go-openvswitch/ovs"
	pb "yunion.io/x/sdnagent/pkg/agent/proto"
)

type openflowService struct {
	pb.UnimplementedOpenflowServer // 嵌入 UnimplementedOpenflowServer
	agent                          *AgentServer
	ofCli                          *ovs.OpenFlowService
}

func newOpenflowService(agent *AgentServer) *openflowService {
	return &openflowService{
		agent: agent,
		ofCli: ovs.New().OpenFlow,
	}
}

func (s *openflowService) newResponse(err error) *pb.Response {
	if err == nil {
		return &pb.Response{Code: 0, Mesg: "ok"}
	}
	return &pb.Response{Code: 1, Mesg: err.Error()}
}

func (s *openflowService) AddFlow(ctx context.Context, in *pb.AddFlowRequest) (*pb.Response, error) {
	f, err := in.Flow.OvsFlow()
	if err != nil {
		err = fmt.Errorf("conversion to ovs.Flow error: %s", err)
		return s.newResponse(err), nil
	}
	flowman := s.agent.GetFlowMan(in.Bridge)
	flowman.AddFlow(ctx, f)
	return s.newResponse(nil), nil
}
func (s *openflowService) DelFlow(ctx context.Context, in *pb.DelFlowRequest) (*pb.Response, error) {
	f, err := in.Flow.OvsFlow()
	if err != nil {
		err = fmt.Errorf("conversion to ovs.Flow error: %s", err)
		return s.newResponse(err), nil
	}
	flowman := s.agent.GetFlowMan(in.Bridge)
	flowman.DelFlow(ctx, f)
	return s.newResponse(nil), nil
}

func (s *openflowService) SyncFlows(ctx context.Context, in *pb.SyncFlowsRequest) (*pb.Response, error) {
	flowman := s.agent.GetFlowMan(in.Bridge)
	flowman.SyncFlows(ctx)
	return s.newResponse(nil), nil
}

func (s *openflowService) DumpBridgePort(ctx context.Context, in *pb.DumpBridgePortRequest) (*pb.DumpBridgePortResponse, error) {
	ofPortStats, err := s.ofCli.DumpPort(in.Bridge, in.Port)
	if err != nil {
		resp := &pb.DumpBridgePortResponse{
			Code: 1,
			Mesg: err.Error(),
		}
		return resp, nil
	}
	resp := &pb.DumpBridgePortResponse{
		Code: 0,
		Mesg: "ok",
		PortStats: &pb.PortStats{
			PortNo: uint32(ofPortStats.PortID),
		},
	}
	return resp, nil
}

func (s *openflowService) DumpFlows(ctx context.Context, in *pb.DumpFlowsRequest) (*pb.DumpFlowsResponse, error) {
	flowman := s.agent.GetFlowMan(in.Bridge)
	flows, err := flowman.DumpFlows(ctx)
	if err != nil {
		resp := &pb.DumpFlowsResponse{
			Code: 1,
			Mesg: err.Error(),
		}
		return resp, nil
	}

	// 转换为 protobuf 消息
	pbFlows := make([]*pb.Flow, len(flows))
	for i, flow := range flows {
		// 直接使用flow的字段构建protobuf消息
		// 这样可以避免解析文本格式的问题
		pbFlows[i] = &pb.Flow{
			Cookie:   flow.Cookie,
			Priority: uint32(flow.Priority),
			Table:    uint32(flow.Table),
			Matches:  "", // 暂时留空，后续可以通过其他方式填充
			Actions:  "", // 暂时留空，后续可以通过其他方式填充
		}

		// 使用MarshalText方法获取完整的流表文本
		txt, err := flow.MarshalText()
		if err != nil {
			resp := &pb.DumpFlowsResponse{
				Code: 1,
				Mesg: fmt.Sprintf("marshal flow error: %s", err),
			}
			return resp, nil
		}

		flowStr := string(txt)

		// 提取actions部分
		actionsIndex := strings.Index(flowStr, "actions=")
		if actionsIndex != -1 {
			actions := flowStr[actionsIndex+8:]
			pbFlows[i].Actions = actions

			// 提取matches部分（不包含actions）
			matches := flowStr[:actionsIndex]
			// 移除cookie、table、priority等字段，只保留匹配条件
			matches = strings.Replace(matches, fmt.Sprintf("cookie=0x%x,", flow.Cookie), "", 1)
			matches = strings.Replace(matches, fmt.Sprintf("table=%d,", flow.Table), "", 1)
			matches = strings.Replace(matches, fmt.Sprintf("priority=%d,", flow.Priority), "", 1)
			// 移除末尾的逗号
			matches = strings.TrimSuffix(matches, ",")
			pbFlows[i].Matches = matches
		}
	}

	resp := &pb.DumpFlowsResponse{
		Code:  0,
		Mesg:  "ok",
		Flows: pbFlows,
	}
	return resp, nil
}
