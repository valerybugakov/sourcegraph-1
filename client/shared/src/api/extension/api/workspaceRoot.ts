import { WorkspaceRoot } from '@sourcegraph/extension-api-types'

import * as sourcegraph from 'sourcegraph'

export class ExtensionWorkspaceRoot implements sourcegraph.WorkspaceRoot {
    public readonly uri: URL
    public readonly inputRevision: string | undefined
    constructor({ uri, inputRevision }: WorkspaceRoot) {
        this.uri = new URL(uri)
        this.inputRevision = inputRevision
    }
}
