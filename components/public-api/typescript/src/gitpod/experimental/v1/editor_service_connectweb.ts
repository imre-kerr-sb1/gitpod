/**
 * Copyright (c) 2023 Gitpod GmbH. All rights reserved.
 * Licensed under the GNU Affero General Public License (AGPL).
 * See License.AGPL.txt in the project root for license information.
 */

// @generated by protoc-gen-connect-web v0.2.1 with parameter "target=ts"
// @generated from file gitpod/experimental/v1/editor_service.proto (package gitpod.experimental.v1, syntax proto3)
/* eslint-disable */
/* @ts-nocheck */

import {ListEditorOptionsRequest, ListEditorOptionsResponse} from "./editor_service_pb.js";
import {MethodKind} from "@bufbuild/protobuf";

/**
 * @generated from service gitpod.experimental.v1.EditorService
 */
export const EditorService = {
  typeName: "gitpod.experimental.v1.EditorService",
  methods: {
    /**
     * @generated from rpc gitpod.experimental.v1.EditorService.ListEditorOptions
     */
    listEditorOptions: {
      name: "ListEditorOptions",
      I: ListEditorOptionsRequest,
      O: ListEditorOptionsResponse,
      kind: MethodKind.Unary,
    },
  }
} as const;
