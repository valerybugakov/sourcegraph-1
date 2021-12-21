import classNames from 'classnames'
import React from 'react'
import { matchPath, Route, RouteProps, Switch, useLocation, useRouteMatch } from 'react-router'
import { NavLink } from 'react-router-dom'

import { CatalogAreaHeader } from './CatalogAreaHeader'
import styles from './CatalogPage.module.scss'

interface Tab extends Pick<RouteProps, 'path' | 'exact'> {
    path: string | string[]
    text: string
    content: React.ReactFragment
}

interface Props {
    path: React.ComponentProps<typeof CatalogAreaHeader>['path']
    tabs: Tab[]
    actions?: React.ReactFragment
}

export const CatalogPage: React.FunctionComponent<Props> = ({ path, tabs, actions }) => {
    const match = useRouteMatch()
    const location = useLocation()
    return (
        <div className="flex-1 d-flex flex-column w-100">
            <CatalogAreaHeader
                path={path}
                nav={
                    <ul className="nav nav-tabs" style={{ marginBottom: '-1px' }}>
                        {tabs.map(({ path, exact, text }) => (
                            <li key={Array.isArray(path) ? path[0] : path} className="nav-item">
                                <NavLink
                                    to={pathWithPrefix(path, match.url)[0]}
                                    isActive={() =>
                                        Boolean(
                                            matchPath(location.pathname, {
                                                path: pathWithPrefix(path, match.url),
                                                exact,
                                            })
                                        )
                                    }
                                    exact={exact}
                                    className={classNames('nav-link px-3', styles.tab)}
                                    data-tab-content={text}
                                >
                                    {text}
                                </NavLink>
                            </li>
                        ))}
                    </ul>
                }
                actions={actions}
            />
            <Switch>
                {tabs.map(({ path, exact, content }) => (
                    <Route
                        key={Array.isArray(path) ? path[0] : path}
                        path={pathWithPrefix(path, match.url)}
                        exact={exact}
                    >
                        {content}
                    </Route>
                ))}
            </Switch>
        </div>
    )
}

function pathWithPrefix(path: string | string[], prefix: string): string[] {
    const paths = Array.isArray(path) ? path : [path]
    return paths.map(path => (path ? `${prefix}/${path}` : prefix))
}