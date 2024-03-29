// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package proto5server

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/internal/fromproto5"
	"github.com/hashicorp/terraform-plugin-framework/internal/fwserver"
	"github.com/hashicorp/terraform-plugin-framework/internal/logging"
	"github.com/hashicorp/terraform-plugin-framework/internal/toproto5"
	"github.com/hashicorp/terraform-plugin-go/tfprotov5"
)

// CallFunction satisfies the tfprotov5.ProviderServer interface.
func (s *Server) CallFunction(ctx context.Context, protoReq *tfprotov5.CallFunctionRequest) (*tfprotov5.CallFunctionResponse, error) {
	ctx = s.registerContext(ctx)
	ctx = logging.InitContext(ctx)

	fwResp := &fwserver.CallFunctionResponse{}

	function, diags := s.FrameworkServer.Function(ctx, protoReq.Name)

	fwResp.Diagnostics.Append(diags...)

	if fwResp.Diagnostics.HasError() {
		return toproto5.CallFunctionResponse(ctx, fwResp), nil
	}

	functionDefinition, diags := s.FrameworkServer.FunctionDefinition(ctx, protoReq.Name)

	fwResp.Diagnostics.Append(diags...)

	if fwResp.Diagnostics.HasError() {
		return toproto5.CallFunctionResponse(ctx, fwResp), nil
	}

	fwReq, diags := fromproto5.CallFunctionRequest(ctx, protoReq, function, functionDefinition)

	fwResp.Diagnostics.Append(diags...)

	if fwResp.Diagnostics.HasError() {
		return toproto5.CallFunctionResponse(ctx, fwResp), nil
	}

	s.FrameworkServer.CallFunction(ctx, fwReq, fwResp)

	return toproto5.CallFunctionResponse(ctx, fwResp), nil
}
