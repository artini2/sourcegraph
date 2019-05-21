import H from 'history'
import React from 'react'
import { ExtensionsControllerProps } from '../../../../../../shared/src/extensions/controller'
import * as GQL from '../../../../../../shared/src/graphql/schema'
import { ThreadSettings } from '../../settings'
import { ThreadSettingsEditForm } from './ThreadSettingsEditForm'

interface Props extends ExtensionsControllerProps {
    thread: GQL.IDiscussionThread
    threadSettings: ThreadSettings
    onThreadUpdate: (thread: GQL.IDiscussionThread) => void
    isLightTheme: boolean
    history: H.History
}

/**
 * The settings page for a single thread.
 */
export const ThreadSettingsPage: React.FunctionComponent<Props> = ({ thread, ...props }) => (
    <div className="thread-settings-page container">
        <ThreadSettingsEditForm {...props} thread={thread} />
    </div>
)