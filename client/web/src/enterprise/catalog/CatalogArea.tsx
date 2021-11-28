import MapSearchIcon from 'mdi-react/MapSearchIcon'
import React, { useEffect } from 'react'
import { Switch, Route, useRouteMatch, RouteComponentProps } from 'react-router'

import { PlatformContextProps } from '@sourcegraph/shared/src/platform/context'
import { SettingsCascadeProps } from '@sourcegraph/shared/src/settings/settings'
import { TelemetryProps } from '@sourcegraph/shared/src/telemetry/telemetryService'

import { AuthenticatedUser } from '../../auth'
import { withAuthenticatedUser } from '../../auth/withAuthenticatedUser'
import { HeroPage } from '../../components/HeroPage'
import { Settings } from '../../schema/settings.schema'

import styles from './CatalogArea.module.scss'
import { useCatalogComponentFilters } from './core/component-filters'
import { ComponentDetailPage } from './pages/component-detail/global/ComponentDetailPage'
import { OverviewPage } from './pages/overview/global/OverviewPage'

const NotFoundPage: React.FunctionComponent = () => <HeroPage icon={MapSearchIcon} title="404: Not Found" />

/**
 * This interface has to receive union type props derived from all child components
 * Because we need to pass all required prop from main Sourcegraph.tsx component to
 * sub-components withing app tree.
 */
export interface CatalogRouterProps extends SettingsCascadeProps<Settings>, PlatformContextProps, TelemetryProps {
    /**
     * Authenticated user info, Used to decide where code insight will appears
     * in personal dashboard (private) or in organisation dashboard (public)
     */
    authenticatedUser: AuthenticatedUser
}

/**
 * The main entrypoint to the catalog UI.
 */
export const CatalogArea = withAuthenticatedUser<CatalogRouterProps>(props => {
    const { platformContext, settingsCascade, telemetryService, authenticatedUser } = props

    const match = useRouteMatch()

    const { filters, onFiltersChange } = useCatalogComponentFilters()

    useEffect(() => () => console.log('DESTROY CatalogArea'), [])

    return (
        <div className={styles.container}>
            <Switch>
                <Route path={match.url} exact={true}>
                    <OverviewPage
                        authenticatedUser={authenticatedUser}
                        filters={filters}
                        onFiltersChange={onFiltersChange}
                        telemetryService={telemetryService}
                    />
                </Route>
                <Route path={`${match.url}/:id`}>
                    {(props: RouteComponentProps<{ id: string }>) => (
                        <ComponentDetailPage
                            key={1}
                            catalogComponentID={props.match.params.id}
                            authenticatedUser={authenticatedUser}
                            filters={filters}
                            onFiltersChange={onFiltersChange}
                            telemetryService={telemetryService}
                        />
                    )}
                </Route>
                <Route component={NotFoundPage} key="hardcoded-key" />
            </Switch>
        </div>
    )
})