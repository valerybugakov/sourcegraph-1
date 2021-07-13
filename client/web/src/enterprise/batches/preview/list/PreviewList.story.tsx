import { boolean } from '@storybook/addon-knobs'
import { storiesOf } from '@storybook/react'
import React, { useContext } from 'react'
import { of, Observable } from 'rxjs'
import { tap } from 'rxjs/operators'

import { BatchSpecApplyPreviewConnectionFields, ChangesetApplyPreviewFields } from '../../../../graphql-operations'
import { EnterpriseWebStory } from '../../../components/EnterpriseWebStory'
import { MultiSelectContext, MultiSelectContextProvider } from '../../MultiSelectContext'
import { BatchChangePreviewContext, BatchChangePreviewContextProvider } from '../BatchChangePreviewContext'

import { hiddenChangesetApplyPreviewStories } from './HiddenChangesetApplyPreviewNode.story'
import { PreviewList } from './PreviewList'
import { visibleChangesetApplyPreviewNodeStories } from './VisibleChangesetApplyPreviewNode.story'

const { add } = storiesOf('web/batches/preview/PreviewList', module)
    .addDecorator(story => <div className="p-3 container">{story()}</div>)
    .addParameters({
        chromatic: {
            viewports: [320, 576, 978, 1440],
        },
    })

const nodes: ChangesetApplyPreviewFields[] = [
    ...Object.values(visibleChangesetApplyPreviewNodeStories),
    ...Object.values(hiddenChangesetApplyPreviewStories),
]

const queryChangesetApplyPreview = (): Observable<BatchSpecApplyPreviewConnectionFields> =>
    of({
        pageInfo: {
            endCursor: null,
            hasNextPage: false,
        },
        totalCount: nodes.length,
        nodes,
    })

const queryEmptyFileDiffs = () => of({ totalCount: 0, pageInfo: { endCursor: null, hasNextPage: false }, nodes: [] })

add('List view', () => (
    <EnterpriseWebStory>
        {props => (
            <MultiSelectContextProvider
                initialVisible={nodes
                    .map(node =>
                        node.targets.__typename === 'VisibleApplyPreviewTargetsAttach' ||
                        node.targets.__typename === 'VisibleApplyPreviewTargetsUpdate'
                            ? node.targets.changesetSpec.id
                            : null
                    )
                    .filter((maybeID): maybeID is string => maybeID !== null)}
            >
                <BatchChangePreviewContextProvider initialHasMorePages={boolean('hasMorePages', true)}>
                    <PreviewList
                        {...props}
                        batchSpecID="123123"
                        authenticatedUser={{
                            url: '/users/alice',
                            displayName: 'Alice',
                            username: 'alice',
                            email: 'alice@email.test',
                        }}
                        queryChangesetApplyPreview={queryChangesetApplyPreview}
                        queryChangesetSpecFileDiffs={queryEmptyFileDiffs}
                        selectionEnabled={boolean('selectionEnabled', false)}
                    />
                </BatchChangePreviewContextProvider>
            </MultiSelectContextProvider>
        )}
    </EnterpriseWebStory>
))
