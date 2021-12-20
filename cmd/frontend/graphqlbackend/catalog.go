package graphqlbackend

import (
	"context"

	"github.com/graph-gophers/graphql-go"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/graphqlbackend/graphqlutil"
)

// This file just contains stub GraphQL resolvers and data types for the Catalog which merely return
// an error if not running in enterprise mode. The actual resolvers are in
// enterprise/cmd/frontend/internal/catalog/resolvers.

// CatalogRootResolver is the root resolver.
type CatalogRootResolver interface {
	Component(context.Context, *ComponentArgs) (ComponentResolver, error)
	Components(context.Context, *CatalogComponentsArgs) (ComponentConnectionResolver, error)
	Graph(context.Context, *CatalogGraphArgs) (CatalogGraphResolver, error)
	Groups() []GroupResolver
	Group(context.Context, *GroupArgs) (GroupResolver, error)
	ComponentTags(context.Context) ([]ComponentTagResolver, error)

	RepositoryComponents(context.Context, *RepositoryResolver, *RepositoryComponentsArgs) ([]ComponentResolver, error)
	GitTreeEntryComponents(context.Context, *GitTreeEntryResolver, *GitTreeEntryComponentsArgs) ([]ComponentResolver, error)

	NodeResolvers() map[string]NodeByIDFunc
}

type ComponentArgs struct {
	Name string
}

type CatalogComponentsArgs struct {
	Query *string
	First *int32
	After *string
}

type CatalogGraphArgs struct {
	Query *string
}

type GroupArgs struct {
	Name string
}

type CatalogGraphResolver interface {
	Nodes() []ComponentResolver
	Edges() []ComponentRelationEdgeResolver
}

type ComponentLifecycle string

type ComponentRelatedEntitiesArgs struct {
	// TODO(sqs): renamed
	Query *string
	First *int32
	After *string
}

type ComponentOwnerResolver struct {
	Person *PersonResolver
	Group  GroupResolver
}

func (r *ComponentOwnerResolver) ToPerson() (*PersonResolver, bool) { return r.Person, r.Person != nil }
func (r *ComponentOwnerResolver) ToGroup() (GroupResolver, bool)    { return r.Group, r.Group != nil }

type GroupResolver interface {
	Node
	Name() string
	Title() string
	Description() *string
	URL() string
	ParentGroup() GroupResolver
	AncestorGroups() []GroupResolver
	ChildGroups() []GroupResolver
	DescendentGroups() []GroupResolver
	Members() []*PersonResolver
	OwnedEntities() []ComponentResolver
}

type ComponentStatusResolver interface {
	ID() graphql.ID
	Contexts() []ComponentStatusContextResolver
	State() ComponentStatusState
}

type ComponentStatusState string

type ComponentStatusContextResolver interface {
	ID() graphql.ID
	Name() string
	State() ComponentStatusState
	Title() string
	Description() *string
	TargetURL() *string
}

type ComponentRelationType string

type ComponentRelationEdgeResolver interface {
	Type() ComponentRelationType
	OutNode() ComponentResolver
	InNode() ComponentResolver
}

type ComponentRelatedEntityConnectionResolver interface {
	Edges() []ComponentRelatedEntityEdgeResolver
}

type ComponentRelatedEntityEdgeResolver interface {
	Node() ComponentResolver
	Type() ComponentRelationType
}

type WhoKnowsArgs struct {
	Query *string
}

type WhoKnowsEdgeResolver interface {
	Node() *PersonResolver
	Reasons() []string
	Score() float64
}

type ComponentConnectionResolver interface {
	Nodes(context.Context) ([]ComponentResolver, error)
	TotalCount(context.Context) (int32, error)
	PageInfo(context.Context) (*graphqlutil.PageInfo, error)
	Tags(context.Context) ([]ComponentTagResolver, error)
}

type ComponentResolver interface {
	ID() graphql.ID
	Name() string
	Description() *string
	Kind() ComponentKind
	Lifecycle() ComponentLifecycle
	Owner(context.Context) (*ComponentOwnerResolver, error)
	Labels(context.Context) ([]ComponentLabelResolver, error)
	Tags(context.Context) ([]ComponentTagResolver, error)
	SourceLocations(context.Context) ([]ComponentSourceLocationResolver, error)
	URL() string
	Status(context.Context) (ComponentStatusResolver, error)

	CodeOwners(context.Context) (*[]ComponentCodeOwnerEdgeResolver, error)
	RelatedEntities(context.Context, *ComponentRelatedEntitiesArgs) (ComponentRelatedEntityConnectionResolver, error)
	WhoKnows(context.Context, *WhoKnowsArgs) ([]WhoKnowsEdgeResolver, error)

	Readme(context.Context) (FileResolver, error)
	Commits(context.Context, *graphqlutil.ConnectionArgs) (GitCommitConnectionResolver, error)
	Branches(context.Context, *graphqlutil.ConnectionArgs) (GitRefConnectionResolver, error)
	Authors(context.Context) (*[]ComponentAuthorEdgeResolver, error)
	Usage(context.Context, *ComponentUsageArgs) (ComponentUsageResolver, error)
}

type ComponentKind string

type ComponentLabelResolver interface {
	Key() string
	Values() []string
}

type ComponentTagResolver interface {
	Name() string
	Components(context.Context, *ComponentTagComponentsArgs) (ComponentConnectionResolver, error)
}

type ComponentSourceLocationResolver interface {
	ID() graphql.ID
	RepositoryName() string
	Repository() (*RepositoryResolver, error)
	Path() *string
	IsEntireRepository() bool
	TreeEntry() (*GitTreeEntryResolver, error)
	IsPrimary() bool
}

type ComponentTagComponentsArgs struct {
	First *int32
	After *string
}

type ComponentAuthorEdgeResolver interface {
	Component() ComponentResolver
	Person() *PersonResolver
	AuthoredLineCount() int32
	AuthoredLineProportion() float64
	LastCommit(context.Context) (*GitCommitResolver, error)
}

type ComponentCodeOwnerEdgeResolver interface {
	Node() *PersonResolver
	FileCount() int32
	FileProportion() float64
}

type ComponentUsageArgs struct {
	Query *string
}

type ComponentUsageResolver interface {
	Locations(context.Context) (LocationConnectionResolver, error)
	People(context.Context) ([]ComponentUsedByPersonEdgeResolver, error)
	Components(context.Context) ([]ComponentUsedByComponentEdgeResolver, error)
}

type ComponentUsedByPersonEdgeResolver interface {
	Node() *PersonResolver
	Locations(context.Context) (LocationConnectionResolver, error)
	AuthoredLineCount() int32
	LastCommit(context.Context) (*GitCommitResolver, error)
}

type ComponentUsedByComponentEdgeResolver interface {
	Node() ComponentResolver
	Locations(context.Context) (LocationConnectionResolver, error)
}

type PackageResolver interface {
	ID() graphql.ID
	Name() string
	URL() string

	TagPackageEntity()
}

type RepositoryComponentsArgs struct {
	Path      string
	Recursive bool
}

func (r *RepositoryResolver) Components(ctx context.Context, args *RepositoryComponentsArgs) ([]ComponentResolver, error) {
	return EnterpriseResolvers.catalogRootResolver.RepositoryComponents(ctx, r, args)
}

type GitTreeEntryComponentsArgs struct{}

func (r *GitTreeEntryResolver) Components(ctx context.Context) ([]ComponentResolver, error) {
	return EnterpriseResolvers.catalogRootResolver.GitTreeEntryComponents(ctx, r, &GitTreeEntryComponentsArgs{})
}
