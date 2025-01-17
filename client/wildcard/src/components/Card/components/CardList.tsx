import classNames from 'classnames'
import React from 'react'

import { ForwardReferenceComponent } from '../../..'

import styles from './CardList.module.scss'

interface CardListProps {}

export const CardList = React.forwardRef(({ as: Component = 'div', children, className, ...attributes }, reference) => (
    <Component ref={reference} className={classNames(styles.listGroup, className)} {...attributes}>
        {children}
    </Component>
)) as ForwardReferenceComponent<'div', CardListProps>
