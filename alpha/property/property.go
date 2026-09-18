package property

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/operator-framework/api/pkg/operators/v1alpha1"
)

type Property struct {
	Type  string          `json:"type"`
	Value json.RawMessage `json:"value"`
}

func (p Property) Validate() error {
	if len(p.Type) == 0 {
		return errors.New("type must be set")
	}
	if len(p.Value) == 0 {
		return errors.New("value must be set")
	}
	var raw json.RawMessage
	if err := json.Unmarshal(p.Value, &raw); err != nil {
		return fmt.Errorf("value is not valid json: %v", err)
	}
	return nil
}

func (p Property) String() string {
	return fmt.Sprintf("type: %q, value: %q", p.Type, p.Value)
}

type Package struct {
	PackageName string `json:"packageName"`
	Version     string `json:"version"`
	Release     string `json:"release,omitzero"`
}

// NOTICE: The Channel properties are for internal use only.
// DO NOT use it for any public-facing functionalities.
type Channel struct {
	ChannelName string `json:"channelName"`
}

type PackageRequired struct {
	PackageName  string `json:"packageName"`
	VersionRange string `json:"versionRange"`
}

type GVK struct {
	Group   string `json:"group"`
	Kind    string `json:"kind"`
	Version string `json:"version"`
}

type GVKRequired struct {
	Group   string `json:"group"`
	Kind    string `json:"kind"`
	Version string `json:"version"`
}

type BundleObject struct {
	Data []byte `json:"data"`
}

// APIServiceDefinitions contains the API service metadata persisted in a catalog.
type APIServiceDefinitions struct {
	Owned    []APIServiceDescription `json:"owned,omitempty"`
	Required []APIServiceDescription `json:"required,omitempty"`
}

// APIServiceDescription contains the API service fields consumed by catalog clients.
type APIServiceDescription struct {
	Name        string `json:"name"`
	Group       string `json:"group"`
	Version     string `json:"version"`
	Kind        string `json:"kind"`
	DisplayName string `json:"displayName,omitempty"`
	Description string `json:"description,omitempty"`
}

// CustomResourceDefinitions contains the CRD metadata persisted in a catalog.
type CustomResourceDefinitions struct {
	Owned    []CRDDescription `json:"owned,omitempty"`
	Required []CRDDescription `json:"required,omitempty"`
}

// CRDDescription contains the CRD fields consumed by catalog clients.
type CRDDescription struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Kind        string `json:"kind"`
	DisplayName string `json:"displayName,omitempty"`
	Description string `json:"description,omitempty"`
}

// CSVMetadata is the canonical, compact CSV metadata property representation.
type CSVMetadata struct {
	Annotations               map[string]string                  `json:"annotations,omitempty"`
	APIServiceDefinitions     APIServiceDefinitions              `json:"apiServiceDefinitions,omitempty"`
	CustomResourceDefinitions CustomResourceDefinitions          `json:"crdDescriptions,omitempty"`
	Description               string                             `json:"description,omitempty"`
	DisplayName               string                             `json:"displayName,omitempty"`
	InstallModes              []v1alpha1.InstallMode             `json:"installModes,omitempty"`
	Keywords                  []string                           `json:"keywords,omitempty"`
	Labels                    map[string]string                  `json:"labels,omitempty"`
	Links                     []v1alpha1.AppLink                 `json:"links,omitempty"`
	Maintainers               []v1alpha1.Maintainer              `json:"maintainers,omitempty"`
	Maturity                  string                             `json:"maturity,omitempty"`
	MinKubeVersion            string                             `json:"minKubeVersion,omitempty"`
	NativeAPIs                []metav1.GroupVersionKind          `json:"nativeAPIs,omitempty"`
	Provider                  v1alpha1.AppLink                   `json:"provider,omitempty"`
}

type Properties struct {
	Packages         []Package         `hash:"set"`
	PackagesRequired []PackageRequired `hash:"set"`
	GVKs             []GVK             `hash:"set"`
	GVKsRequired     []GVKRequired     `hash:"set"`
	BundleObjects    []BundleObject    `hash:"set"`
	Channels         []Channel         `hash:"set"`
	CSVMetadatas     []CSVMetadata     `hash:"set"`

	Others []Property `hash:"set"`
}

const (
	TypePackage         = "olm.package"
	TypePackageRequired = "olm.package.required"
	TypeGVK             = "olm.gvk"
	TypeGVKRequired     = "olm.gvk.required"
	TypeBundleObject    = "olm.bundle.object"
	TypeCSVMetadata     = "olm.csv.metadata"
	TypeConstraint      = "olm.constraint"
	TypeChannel         = "olm.channel"
)

func Parse(in []Property) (*Properties, error) {
	var out Properties
	for i, prop := range in {
		switch prop.Type {
		case TypePackage:
			var p Package
			if err := json.Unmarshal(prop.Value, &p); err != nil {
				return nil, ParseError{Idx: i, Typ: prop.Type, Err: err}
			}
			out.Packages = append(out.Packages, p)
		case TypePackageRequired:
			var p PackageRequired
			if err := json.Unmarshal(prop.Value, &p); err != nil {
				return nil, ParseError{Idx: i, Typ: prop.Type, Err: err}
			}
			out.PackagesRequired = append(out.PackagesRequired, p)
		case TypeGVK:
			var p GVK
			if err := json.Unmarshal(prop.Value, &p); err != nil {
				return nil, ParseError{Idx: i, Typ: prop.Type, Err: err}
			}
			out.GVKs = append(out.GVKs, p)
		case TypeGVKRequired:
			var p GVKRequired
			if err := json.Unmarshal(prop.Value, &p); err != nil {
				return nil, ParseError{Idx: i, Typ: prop.Type, Err: err}
			}
			out.GVKsRequired = append(out.GVKsRequired, p)
		case TypeBundleObject:
			var p BundleObject
			if err := json.Unmarshal(prop.Value, &p); err != nil {
				return nil, ParseError{Idx: i, Typ: prop.Type, Err: err}
			}
			out.BundleObjects = append(out.BundleObjects, p)
		case TypeCSVMetadata:
			var p CSVMetadata
			if err := json.Unmarshal(prop.Value, &p); err != nil {
				return nil, ParseError{Idx: i, Typ: prop.Type, Err: err}
			}
			out.CSVMetadatas = append(out.CSVMetadatas, p)
		// NOTICE: The Channel properties are for internal use only.
		//   DO NOT use it for any public-facing functionalities.
		//   This API is in alpha stage and it is subject to change.
		case TypeChannel:
			var p Channel
			if err := json.Unmarshal(prop.Value, &p); err != nil {
				return nil, ParseError{Idx: i, Typ: prop.Type, Err: err}
			}
			out.Channels = append(out.Channels, p)
		default:
			var p json.RawMessage
			if err := json.Unmarshal(prop.Value, &p); err != nil {
				return nil, ParseError{Idx: i, Typ: prop.Type, Err: err}
			}
			out.Others = append(out.Others, prop)
		}
	}
	return &out, nil
}

func Deduplicate(in []Property) []Property {
	type key struct {
		typ   string
		value string
	}

	props := map[key]Property{}
	// nolint:prealloc
	var out []Property
	for _, p := range in {
		k := key{p.Type, string(p.Value)}
		if _, ok := props[k]; ok {
			continue
		}
		props[k] = p
		out = append(out, p)
	}
	return out
}

func Build(p interface{}) (*Property, error) {
	var (
		typ string
		val interface{}
	)
	if prop, ok := p.(*Property); ok {
		typ = prop.Type
		val = prop.Value
	} else {
		t := reflect.TypeOf(p)
		if t.Kind() != reflect.Ptr {
			return nil, errors.New("input must be a pointer to a type")
		}
		typ, ok = scheme[t]
		if !ok {
			return nil, fmt.Errorf("%s not a known property type registered with the scheme", t)
		}
		val = p
	}
	d, err := jsonMarshal(val)
	if err != nil {
		return nil, err
	}

	return &Property{
		Type:  typ,
		Value: d,
	}, nil
}

func MustBuild(p interface{}) Property {
	prop, err := Build(p)
	if err != nil {
		panic(err)
	}
	return *prop
}

func jsonMarshal(p interface{}) ([]byte, error) {
	buf := &bytes.Buffer{}
	dec := json.NewEncoder(buf)
	dec.SetEscapeHTML(false)
	err := dec.Encode(p)
	if err != nil {
		return nil, err
	}
	out := &bytes.Buffer{}
	if err := json.Compact(out, buf.Bytes()); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

func MustBuildPackage(name, version string) Property {
	return MustBuild(&Package{PackageName: name, Version: version})
}
func MustBuildPackageRelease(name, version, relVersion string) Property {
	return MustBuild(&Package{PackageName: name, Version: version, Release: relVersion})
}
func MustBuildPackageRequired(name, versionRange string) Property {
	return MustBuild(&PackageRequired{name, versionRange})
}
func MustBuildGVK(group, version, kind string) Property {
	return MustBuild(&GVK{group, kind, version})
}
func MustBuildGVKRequired(group, version, kind string) Property {
	return MustBuild(&GVKRequired{group, kind, version})
}
func MustBuildBundleObject(data []byte) Property {
	return MustBuild(&BundleObject{Data: data})
}

func MustBuildCSVMetadata(csv v1alpha1.ClusterServiceVersion) Property {
	return MustBuild(&CSVMetadata{
		Annotations:               csv.GetAnnotations(),
		APIServiceDefinitions:     newAPIServiceDefinitions(csv.Spec.APIServiceDefinitions),
		CustomResourceDefinitions: newCustomResourceDefinitions(csv.Spec.CustomResourceDefinitions),
		Description:               csv.Spec.Description,
		DisplayName:               csv.Spec.DisplayName,
		InstallModes:              csv.Spec.InstallModes,
		Keywords:                  csv.Spec.Keywords,
		Labels:                    csv.GetLabels(),
		Links:                     csv.Spec.Links,
		Maintainers:               csv.Spec.Maintainers,
		Maturity:                  csv.Spec.Maturity,
		MinKubeVersion:            csv.Spec.MinKubeVersion,
		NativeAPIs:                csv.Spec.NativeAPIs,
		Provider:                  csv.Spec.Provider,
	})
}

func newAPIServiceDefinitions(apis v1alpha1.APIServiceDefinitions) APIServiceDefinitions {
	return APIServiceDefinitions{
		Owned:    newAPIServices(apis.Owned),
		Required: newAPIServices(apis.Required),
	}
}

func newAPIServices(descriptions []v1alpha1.APIServiceDescription) []APIServiceDescription {
	if descriptions == nil {
		return nil
	}
	services := make([]APIServiceDescription, len(descriptions))
	for i, description := range descriptions {
		services[i] = APIServiceDescription{
			Name:        description.Name,
			Group:       description.Group,
			Version:     description.Version,
			Kind:        description.Kind,
			DisplayName: description.DisplayName,
			Description: description.Description,
		}
	}
	return services
}

func newCustomResourceDefinitions(crds v1alpha1.CustomResourceDefinitions) CustomResourceDefinitions {
	return CustomResourceDefinitions{
		Owned:    newCRDs(crds.Owned),
		Required: newCRDs(crds.Required),
	}
}

func newCRDs(descriptions []v1alpha1.CRDDescription) []CRDDescription {
	if descriptions == nil {
		return nil
	}
	crds := make([]CRDDescription, len(descriptions))
	for i, description := range descriptions {
		crds[i] = CRDDescription{
			Name:        description.Name,
			Version:     description.Version,
			Kind:        description.Kind,
			DisplayName: description.DisplayName,
			Description: description.Description,
		}
	}
	return crds
}

// ToV1Alpha1 converts compact API service metadata to the upstream API type.
func (d APIServiceDefinitions) ToV1Alpha1() v1alpha1.APIServiceDefinitions {
	return v1alpha1.APIServiceDefinitions{
		Owned:    d.apiServicesToV1Alpha1(d.Owned),
		Required: d.apiServicesToV1Alpha1(d.Required),
	}
}

func (d APIServiceDefinitions) apiServicesToV1Alpha1(descriptions []APIServiceDescription) []v1alpha1.APIServiceDescription {
	if descriptions == nil {
		return nil
	}
	services := make([]v1alpha1.APIServiceDescription, len(descriptions))
	for i, description := range descriptions {
		services[i] = v1alpha1.APIServiceDescription{
			Name:        description.Name,
			Group:       description.Group,
			Version:     description.Version,
			Kind:        description.Kind,
			DisplayName: description.DisplayName,
			Description: description.Description,
		}
	}
	return services
}

// ToV1Alpha1 converts compact CRD metadata to the upstream API type.
func (d CustomResourceDefinitions) ToV1Alpha1() v1alpha1.CustomResourceDefinitions {
	return v1alpha1.CustomResourceDefinitions{
		Owned:    d.crdsToV1Alpha1(d.Owned),
		Required: d.crdsToV1Alpha1(d.Required),
	}
}

func (d CustomResourceDefinitions) crdsToV1Alpha1(descriptions []CRDDescription) []v1alpha1.CRDDescription {
	if descriptions == nil {
		return nil
	}
	crds := make([]v1alpha1.CRDDescription, len(descriptions))
	for i, description := range descriptions {
		crds[i] = v1alpha1.CRDDescription{
			Name:        description.Name,
			Version:     description.Version,
			Kind:        description.Kind,
			DisplayName: description.DisplayName,
			Description: description.Description,
		}
	}
	return crds
}

// CanonicalizeCSVMetadataProperty decodes legacy fields and rebuilds compact JSON.
func CanonicalizeCSVMetadataProperty(p Property) (Property, error) {
	var metadata CSVMetadata
	if err := json.Unmarshal(p.Value, &metadata); err != nil {
		return Property{}, err
	}
	canonical, err := Build(&metadata)
	if err != nil {
		return Property{}, err
	}
	return *canonical, nil
}
