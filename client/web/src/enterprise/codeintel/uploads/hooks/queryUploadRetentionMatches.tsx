import { ApolloClient } from '@apollo/client'
import { from, Observable } from 'rxjs'
import { map } from 'rxjs/operators'

import { getDocumentNode, gql } from '@sourcegraph/http-client'
import { IRetentionPolicyOverviewOnLSIFUploadArguments } from '@sourcegraph/shared/src/schema'

import { Connection } from '../../../../components/FilteredConnection'
import {
    GitObjectType,
    LsifUploadRetentionMatchesResult,
    LsifUploadRetentionMatchesVariables,
} from '../../../../graphql-operations'
import { retentionByUploadTitle } from '../components/UploadRetentionStatusNode'

interface UploadRetentionMatchesResult {
    node: {
        retentionPolicyOverview: {
            nodes: {
                matches: boolean
                protectingCommits: string[] | null
                configurationPolicy: {
                    id: string
                    name: string
                    type: GitObjectType
                    retentionDurationHours: number | null
                }
            }[]
            totalCount: number | null
            pageInfo: { endCursor: string | null; hasNextPage: boolean }
        }
    }
    lsifUploads: {
        nodes: {
            id: string
            inputCommit: string
            inputRoot: string
            projectRoot: {
                repository: { id: string; name: string }
            } | null
        }[]
        totalCount: number | null
    }
}

export type NormalizedUploadRetentionMatch =
    | {
          matchType: 'RetentionPolicy'
          matches: boolean
          protectingCommits: string[]
          configurationPolicy: {
              id: string
              name: string
              type: GitObjectType
              retentionDurationHours: number | null
          }
      }
    | {
          matchType: 'UploadReference'
          uploadSlice: {
              id: string
              inputCommit: string
              inputRoot: string
              projectRoot: {
                  repository: { id: string; name: string }
              } | null
          }[]
          total: number
      }

const UPLOAD_RETENTIONS_QUERY = gql`
    query LsifUploadRetentionMatches(
        $id: ID!
        $matchesOnly: Boolean!
        $after: String
        $first: Int
        $query: String
    ) {
        node(id: $id) {
            __typename
            ... on LSIFUpload {
                retentionPolicyOverview(matchesOnly: $matchesOnly, query: $query, after: $after, first: $first) {
                    __typename
                    nodes {
                        __typename
                        # id
                        configurationPolicy {
                            __typename
                            id
                            name
                            type
                            retentionDurationHours
                        }
                        matches
                        protectingCommits
                    }
                    totalCount
                    pageInfo {
                        endCursor
                        hasNextPage
                    }
                }
            }
        }

        lsifUploads(dependentOf: $id) {
            __typename
            totalCount
            nodes {
                id
                inputCommit
                inputRoot
                projectRoot {
                    repository {
                        name
                        id
                    }
                }
            }
        }
    }
`
export const queryUploadRetentionMatches = (
    client: ApolloClient<object>,
    id: string,
    { matchesOnly, after, first, query }: IRetentionPolicyOverviewOnLSIFUploadArguments
): Observable<Connection<NormalizedUploadRetentionMatch>> => {
    const vars: LsifUploadRetentionMatchesVariables = {
        id,
        matchesOnly,
        query: query ?? null,
        first: first ?? null,
        after: after ?? null,
    }

    return from(
        client.query<LsifUploadRetentionMatchesResult, LsifUploadRetentionMatchesVariables>({
            query: getDocumentNode(UPLOAD_RETENTIONS_QUERY),
            variables: { ...vars },
        })
    ).pipe(
        map(({ data }) => {
            const { node, ...rest } = data
            if (!node || node.__typename !== 'LSIFUpload') {
                throw new Error('No such LSIFUpload')
            }

            const matchesResult: UploadRetentionMatchesResult = { node, ...rest }
            return matchesResult
        }),
        map(({ node, lsifUploads }) => {
            const conn: Connection<NormalizedUploadRetentionMatch> = {
                totalCount: (node.retentionPolicyOverview.totalCount ?? 0) + (lsifUploads.totalCount ?? 0) > 0 ? 1 : 0,
                nodes: [],
            }

            if ((lsifUploads.totalCount ?? 0) > 0 && retentionByUploadTitle.toLowerCase().includes(query ?? '')) {
                conn.nodes.push({
                    matchType: 'UploadReference',
                    uploadSlice: lsifUploads.nodes,
                    total: lsifUploads.totalCount ?? 0,
                })
            }

            conn.nodes.push(
                ...node.retentionPolicyOverview.nodes.map(
                    (node): NormalizedUploadRetentionMatch => ({
                        matchType: 'RetentionPolicy',
                        ...node,
                        protectingCommits: node.protectingCommits ?? [],
                    })
                )
            )

            console.log(JSON.stringify(conn.nodes, null, 4))

            conn.pageInfo = node.retentionPolicyOverview.pageInfo

            return conn
        })
    )
}
