import React from 'react'
import { combineLatest, Observable } from 'rxjs'
import { catchError, map, switchMap } from 'rxjs/operators'
import * as sourcegraph from 'sourcegraph'
import { ContributableViewContainer, TextDocumentPositionParams } from '../../protocol'
import * as plain from '../../protocol/plainTypes'
import { ProvideTextDocumentLocationSignature } from './location'
import { Entry, FeatureProviderRegistry } from './registry'

export interface ViewProviderRegistrationOptions {
    id: string
    container: ContributableViewContainer
}

export interface PanelViewWithComponent extends Pick<sourcegraph.PanelView, 'title' | 'content'> {
    /**
     * The location provider whose results to render in the panel view.
     */
    // TODO!(sqs): why null and not ?:
    locationProvider: ProvideTextDocumentLocationSignature<TextDocumentPositionParams, plain.Location> | null

    /**
     * The React element to render in the panel view.
     */
    reactElement?: React.ReactFragment
}

export type ProvideViewSignature = Observable<PanelViewWithComponent | null>

/** Provides views from all extensions. */
export class ViewProviderRegistry extends FeatureProviderRegistry<
    ViewProviderRegistrationOptions,
    ProvideViewSignature
> {
    /**
     * Returns an observable that emits the specified view whenever it or the set of registered view providers
     * changes. If the provider emits an error, the returned observable also emits an error (and completes).
     */
    public getView(id: string): Observable<PanelViewWithComponent | null> {
        return getView(this.entries, id)
    }

    /**
     * Returns an observable that emits all views whenever the set of registered view providers or their properties
     * change. If any provider emits an error, the error is logged and the provider is omitted from the emission of
     * the observable (the observable does not emit the error).
     */
    public getViews(
        container: ContributableViewContainer
    ): Observable<(PanelViewWithComponent & ViewProviderRegistrationOptions)[]> {
        return getViews(this.entries, container)
    }
}

/*
                {
                    // References panel view.
                    registrationOptions: { id: 'references', container: ContributableViewContainer.Panel },
                    provider: subjectChanges.pipe(
                        map((subject: PanelSubject) => ({
                            title: 'References',
                            content: '',
                            locationProvider: null,
                            reactElement: (
                                <FileLocations
                                    className="panel__tabs-content"
                                    query={this.queryReferencesLocal}
                                    icon={RepositoryIcon}
                                    pluralNoun="local references"
                                    isLightTheme={this.props.isLightTheme}
                                    fetchHighlightedFileLines={fetchHighlightedFileLines}
                                />
                            ),
                        }))
                    ),
                },*/

/**
 * Exported for testing only. Most callers should use {@link ViewProviderRegistry#getView}, which uses the
 * registered view providers.
 *
 * @internal
 */
export function getView(
    entries: Observable<Entry<ViewProviderRegistrationOptions, Observable<PanelViewWithComponent | null>>[]>,
    id: string
): Observable<(PanelViewWithComponent & ViewProviderRegistrationOptions) | null> {
    return entries.pipe(
        map(entries => entries.find(entry => entry.registrationOptions.id === id)),
        switchMap(entry => (entry ? addRegistrationOptions(entry) : [null]))
    )
}

/**
 * Exported for testing only. Most callers should use {@link ViewProviderRegistry#getViews}, which uses the
 * registered view providers.
 *
 * @internal
 */
export function getViews(
    entries: Observable<Entry<ViewProviderRegistrationOptions, Observable<PanelViewWithComponent | null>>[]>,
    container: ContributableViewContainer
): Observable<(PanelViewWithComponent & ViewProviderRegistrationOptions)[]> {
    return entries.pipe(
        switchMap(
            entries =>
                entries && entries.length > 0
                    ? combineLatest(
                          entries.filter(e => e.registrationOptions.container === container).map(entry =>
                              addRegistrationOptions(entry).pipe(
                                  catchError(err => {
                                      console.error(err)
                                      return [null]
                                  })
                              )
                          )
                      ).pipe(
                          map(entries =>
                              entries.filter(
                                  (result): result is PanelViewWithComponent & ViewProviderRegistrationOptions =>
                                      result !== null
                              )
                          )
                      )
                    : [[]]
        )
    )
}

function addRegistrationOptions(
    entry: Entry<ViewProviderRegistrationOptions, Observable<PanelViewWithComponent | null>>
): Observable<(PanelViewWithComponent & ViewProviderRegistrationOptions) | null> {
    return entry.provider.pipe(map(view => view && { ...view, ...entry.registrationOptions }))
}
