package composite

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"testing"

	"github.com/operator-framework/operator-registry/pkg/image"
	"github.com/stretchr/testify/require"
)

var testCompositeFormat = []byte(`
- name: test-catalog
  destination:
    path: my-package
  strategy:
    name: custom
    template:
      schema: olm.builder.custom
      config:
        command: cat
        args:
          - "components/v4.13.yaml"
        output: catalog.yaml
`)

var semverContribution = []byte(`---
schema: olm.semver
stable:
  bundles:
  - image: quay.io/openshift-community-operators/community-kubevirt-hyperconverged:v1.4.0
  - image: quay.io/openshift-community-operators/community-kubevirt-hyperconverged:v1.4.1
  - image: quay.io/openshift-community-operators/community-kubevirt-hyperconverged:v1.4.2
  - image: quay.io/openshift-community-operators/community-kubevirt-hyperconverged:v1.4.3
  - image: quay.io/openshift-community-operators/community-kubevirt-hyperconverged:v1.4.4
  - image: quay.io/openshift-community-operators/community-kubevirt-hyperconverged:v1.5.0
  - image: quay.io/openshift-community-operators/community-kubevirt-hyperconverged:v1.5.1
  - image: quay.io/openshift-community-operators/community-kubevirt-hyperconverged:v1.5.2
  - image: quay.io/openshift-community-operators/community-kubevirt-hyperconverged:v1.6.0
  - image: quay.io/openshift-community-operators/community-kubevirt-hyperconverged:v1.7.0
  - image: quay.io/openshift-community-operators/community-kubevirt-hyperconverged:v1.8.0
  - image: quay.io/openshift-community-operators/community-kubevirt-hyperconverged:v1.8.1
`)

var basicContribution = []byte(`---
---
schema: olm.package
name: kubevirt-hyperconverged
defaultChannel: stable
---
schema: olm.channel
package: kubevirt-hyperconverged
name: stable
entries:
- name: kubevirt-hyperconverged-operator.v4.10.0
- name: kubevirt-hyperconverged-operator.v4.10.1
  replaces: kubevirt-hyperconverged-operator.v4.10.0
- name: kubevirt-hyperconverged-operator.v4.10.2
  replaces: kubevirt-hyperconverged-operator.v4.10.1
- name: kubevirt-hyperconverged-operator.v4.10.3
  replaces: kubevirt-hyperconverged-operator.v4.10.2
---
schema: olm.bundle
image: registry.redhat.io/container-native-virtualization/hco-bundle-registry@sha256:35b29e8eb48d9818a1217d5b89e4dcb7a900c5c5e6ae3745683813c5708c86e9
---
schema: olm.bundle
image: registry.redhat.io/container-native-virtualization/hco-bundle-registry@sha256:ac8b60a91411c0fcc4ab2c52db8b6e7682ee747c8969dde9ad8d1a5aa7d44772
---
schema: olm.bundle
image: registry.redhat.io/container-native-virtualization/hco-bundle-registry@sha256:9c10a5c4e5ffad632bc8b4289fe2bc0c181bc7c4c8270a356ac21ebff568a45e
---
schema: olm.bundle
image: registry.redhat.io/container-native-virtualization/hco-bundle-registry@sha256:34cf9e0dd3cc07c487b364c30b3f95bf352be8ca4fe89d473fc624ad7283651d
`)

var _ Builder = &TestBuilder{}

var TestBuilderSchema = "olm.builder.test"

type TestBuilder struct {
	buildShouldError    bool
	validateShouldError bool
}

func (tb *TestBuilder) Build(ctx context.Context, reg image.Registry, dir string, td TemplateDefinition) error {
	if tb.buildShouldError {
		return fmt.Errorf("build error!")
	}
	return nil
}

func (tb *TestBuilder) Validate(dir string) error {
	if tb.validateShouldError {
		return fmt.Errorf("validate error!")
	}
	return nil
}

var renderValidCatalog = `
schema: olm.composite.catalogs
catalogs:
  - name: first-catalog
    destination:
      baseImage: quay.io/operator-framework/opm:latest
      workingDir: contributions/first-catalog
    builders:
      - olm.builder.test
`

var renderValidComposite = `
schema: olm.composite
components:
  - name: first-catalog
    destination:
      path: my-operator
    strategy:
      name: test
      template:
        schema: olm.builder.test
        config:
          input: components/contribution1.yaml
          output: catalog.yaml
`

func TestCompositeRender(t *testing.T) {
	type testCase struct {
		name              string
		compositeTemplate Template
		validate          bool
		assertions        func(t *testing.T, err error)
	}

	testCases := []testCase{
		{
			name:     "successful render",
			validate: true,
			compositeTemplate: Template{
				catalogFile:      bytes.NewReader([]byte(renderValidCatalog)),
				contributionFile: bytes.NewReader([]byte(renderValidComposite)),
				registeredBuilders: map[string]builderFunc{
					TestBuilderSchema: func(bc BuilderConfig) Builder { return &TestBuilder{} },
				},
			},
			assertions: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		{
			name:     "Builder.Build() failure",
			validate: true,
			compositeTemplate: Template{
				catalogFile:      bytes.NewReader([]byte(renderValidCatalog)),
				contributionFile: bytes.NewReader([]byte(renderValidComposite)),
				registeredBuilders: map[string]builderFunc{
					TestBuilderSchema: func(bc BuilderConfig) Builder { return &TestBuilder{buildShouldError: true} },
				},
			},
			assertions: func(t *testing.T, err error) {
				require.Error(t, err)
				require.Equal(t, "building component \"first-catalog\": build error!", err.Error())
			},
		},
		{
			name:     "Builder.Validate() failure",
			validate: true,
			compositeTemplate: Template{
				catalogFile:      bytes.NewReader([]byte(renderValidCatalog)),
				contributionFile: bytes.NewReader([]byte(renderValidComposite)),
				registeredBuilders: map[string]builderFunc{
					TestBuilderSchema: func(bc BuilderConfig) Builder { return &TestBuilder{validateShouldError: true} },
				},
			},
			assertions: func(t *testing.T, err error) {
				require.Error(t, err)
				require.Equal(t, "validating component \"first-catalog\": validate error!", err.Error())
			},
		},
		// TODO: Cover more cases (likely just adapting below cases to look similar to above cases)
		// {
		// 	name:     "component not in catalog config",
		// 	validate: true,
		// 	compositeTemplate: Template{
		// 		CatalogBuilders: CatalogBuilderMap{
		// 			"testcatalog": BuilderMap{
		// 				"olm.builder.test": &TestBuilder{},
		// 			},
		// 		},
		// 	},
		// 	compositeCfg: CompositeConfig{
		// 		Schema: CompositeSchema,
		// 		Components: []Component{
		// 			{
		// 				Name: "invalid",
		// 				Destination: ComponentDestination{
		// 					Path: "testcatalog/mypackage",
		// 				},
		// 				Strategy: BuildStrategy{
		// 					Name: "testbuild",
		// 					Template: TemplateDefinition{
		// 						Schema: "olm.builder.test",
		// 						Config: json.RawMessage{},
		// 					},
		// 				},
		// 			},
		// 		},
		// 	},
		// 	assertions: func(t *testing.T, err error) {
		// 		require.Error(t, err)
		// 		expectedErr := fmt.Sprintf("building component %q: component does not exist in the catalog configuration. Available components are: %s", "invalid", []string{"testcatalog"})
		// 		require.Equal(t, expectedErr, err.Error())
		// 	},
		// },
		// {
		// 	name:     "builder not in catalog config",
		// 	validate: true,
		// 	compositeTemplate: Template{
		// 		CatalogBuilders: CatalogBuilderMap{
		// 			"testcatalog": BuilderMap{
		// 				"olm.builder.test": &TestBuilder{},
		// 			},
		// 		},
		// 	},
		// 	compositeCfg: CompositeConfig{
		// 		Schema: CompositeSchema,
		// 		Components: []Component{
		// 			{
		// 				Name: "testcatalog",
		// 				Destination: ComponentDestination{
		// 					Path: "testcatalog/mypackage",
		// 				},
		// 				Strategy: BuildStrategy{
		// 					Name: "testbuild",
		// 					Template: TemplateDefinition{
		// 						Schema: "olm.builder.invalid",
		// 						Config: json.RawMessage{},
		// 					},
		// 				},
		// 			},
		// 		},
		// 	},
		// 	assertions: func(t *testing.T, err error) {
		// 		require.Error(t, err)
		// 		expectedErr := fmt.Sprintf("building component %q: no builder found for template schema %q", "testcatalog", "olm.builder.invalid")
		// 		require.Equal(t, expectedErr, err.Error())
		// 	},
		// },
		// {
		// 	name:     "build step error",
		// 	validate: true,
		// 	compositeTemplate: Template{
		// 		CatalogBuilders: CatalogBuilderMap{
		// 			"testcatalog": BuilderMap{
		// 				"olm.builder.test": &TestBuilder{buildError: true},
		// 			},
		// 		},
		// 	},
		// 	compositeCfg: CompositeConfig{
		// 		Schema: CompositeSchema,
		// 		Components: []Component{
		// 			{
		// 				Name: "testcatalog",
		// 				Destination: ComponentDestination{
		// 					Path: "testcatalog/mypackage",
		// 				},
		// 				Strategy: BuildStrategy{
		// 					Name: "testbuild",
		// 					Template: TemplateDefinition{
		// 						Schema: "olm.builder.test",
		// 						Config: json.RawMessage{},
		// 					},
		// 				},
		// 			},
		// 		},
		// 	},
		// 	assertions: func(t *testing.T, err error) {
		// 		require.Error(t, err)
		// 		expectedErr := fmt.Sprintf("building component %q: %s", "testcatalog", buildErr)
		// 		require.Equal(t, expectedErr, err.Error())
		// 	},
		// },
		// {
		// 	name:     "validate step error",
		// 	validate: true,
		// 	compositeTemplate: Template{
		// 		CatalogBuilders: CatalogBuilderMap{
		// 			"testcatalog": BuilderMap{
		// 				"olm.builder.test": &TestBuilder{validateError: true},
		// 			},
		// 		},
		// 	},
		// 	compositeCfg: CompositeConfig{
		// 		Schema: CompositeSchema,
		// 		Components: []Component{
		// 			{
		// 				Name: "testcatalog",
		// 				Destination: ComponentDestination{
		// 					Path: "testcatalog/mypackage",
		// 				},
		// 				Strategy: BuildStrategy{
		// 					Name: "testbuild",
		// 					Template: TemplateDefinition{
		// 						Schema: "olm.builder.test",
		// 						Config: json.RawMessage{},
		// 					},
		// 				},
		// 			},
		// 		},
		// 	},
		// 	assertions: func(t *testing.T, err error) {
		// 		require.Error(t, err)
		// 		expectedErr := fmt.Sprintf("validating component %q: %s", "testcatalog", validateErr)
		// 		require.Equal(t, expectedErr, err.Error())
		// 	},
		// },
		// {
		// 	name:     "validation step skipped",
		// 	validate: false,
		// 	compositeTemplate: Template{
		// 		CatalogBuilders: CatalogBuilderMap{
		// 			"testcatalog": BuilderMap{
		// 				"olm.builder.test": &TestBuilder{validateError: true},
		// 			},
		// 		},
		// 	},
		// 	compositeCfg: CompositeConfig{
		// 		Schema: CompositeSchema,
		// 		Components: []Component{
		// 			{
		// 				Name: "testcatalog",
		// 				Destination: ComponentDestination{
		// 					Path: "testcatalog/mypackage",
		// 				},
		// 				Strategy: BuildStrategy{
		// 					Name: "testbuild",
		// 					Template: TemplateDefinition{
		// 						Schema: "olm.builder.test",
		// 						Config: json.RawMessage{},
		// 					},
		// 				},
		// 			},
		// 		},
		// 	},
		// 	assertions: func(t *testing.T, err error) {
		// 		// the validate step would error but since
		// 		// we are skipping it we expect no error to occur
		// 		require.NoError(t, err)
		// 	},
		// },
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.compositeTemplate.Render(context.Background(), tc.validate)
			tc.assertions(t, err)
		})
	}
}

func TestBuilderForSchema(t *testing.T) {
	type testCase struct {
		name          string
		builderSchema string
		builderCfg    BuilderConfig
		assertions    func(t *testing.T, builder Builder, err error)
	}

	testCases := []testCase{
		{
			name:          "Basic Builder Schema",
			builderSchema: BasicBuilderSchema,
			builderCfg:    BuilderConfig{},
			assertions: func(t *testing.T, builder Builder, err error) {
				require.NoError(t, err)
				require.IsType(t, &BasicBuilder{}, builder)
			},
		},
		{
			name:          "Semver Builder Schema",
			builderSchema: SemverBuilderSchema,
			builderCfg:    BuilderConfig{},
			assertions: func(t *testing.T, builder Builder, err error) {
				require.NoError(t, err)
				require.IsType(t, &SemverBuilder{}, builder)
			},
		},
		{
			name:          "Raw Builder Schema",
			builderSchema: RawBuilderSchema,
			builderCfg:    BuilderConfig{},
			assertions: func(t *testing.T, builder Builder, err error) {
				require.NoError(t, err)
				require.IsType(t, &RawBuilder{}, builder)
			},
		},
		{
			name:          "Custom Builder Schema",
			builderSchema: CustomBuilderSchema,
			builderCfg:    BuilderConfig{},
			assertions: func(t *testing.T, builder Builder, err error) {
				require.NoError(t, err)
				require.IsType(t, &CustomBuilder{}, builder)
			},
		},
		{
			name:          "Invalid Builder Schema",
			builderSchema: "invalid",
			builderCfg:    BuilderConfig{},
			assertions: func(t *testing.T, builder Builder, err error) {
				require.Error(t, err)
				require.Equal(t, fmt.Sprintf("unknown schema %q", "invalid"), err.Error())
				require.Nil(t, builder)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			template := NewTemplate()
			builder, err := template.builderForSchema(tc.builderSchema, tc.builderCfg)
			tc.assertions(t, builder, err)
		})
	}

}

var validCatalog = `
schema: olm.composite.catalogs
catalogs:
  - name: first-catalog
    destination:
      baseImage: quay.io/operator-framework/opm:latest
      workingDir: contributions/first-catalog
    builders:
      - olm.builder.semver
      - olm.builder.basic
  - name: second-catalog
    destination:
      baseImage: quay.io/operator-framework/opm:latest
      workingDir: contributions/second-catalog
    builders:
      - olm.builder.semver
  - name: test-catalog
    destination:
      baseImage: quay.io/operator-framework/opm:latest
      workingDir: contributions/test-catalog
    builders:
      - olm.builder.custom`

var unmarshalFail = `
invalid
`

var invalidSchemaCatalog = `
schema: invalid
catalogs:
  - name: first-catalog
    destination:
      baseImage: quay.io/operator-framework/opm:latest
      workingDir: contributions/first-catalog
    builders:
      - olm.builder.semver
      - olm.builder.basic
`

func TestParseCatalogSpec(t *testing.T) {
	type testCase struct {
		name       string
		catalog    []byte
		assertions func(t *testing.T, catalog *CatalogConfig, err error)
	}

	testCases := []testCase{
		{
			name:    "Valid catalog configuration",
			catalog: []byte(validCatalog),
			assertions: func(t *testing.T, catalog *CatalogConfig, err error) {
				require.NoError(t, err)
				require.Equal(t, 3, len(catalog.Catalogs))
			},
		},
		{
			name:    "Unmarshal failure",
			catalog: []byte(unmarshalFail),
			assertions: func(t *testing.T, catalog *CatalogConfig, err error) {
				require.Error(t, err)
				require.Equal(t, "unmarshalling catalog config: json: cannot unmarshal string into Go value of type composite.CatalogConfig", err.Error())
			},
		},
		{
			name:    "Invalid schema",
			catalog: []byte(invalidSchemaCatalog),
			assertions: func(t *testing.T, catalog *CatalogConfig, err error) {
				require.Error(t, err)
				require.Equal(t, fmt.Sprintf("catalog configuration file has unknown schema, should be %q", CatalogSchema), err.Error())
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			template := NewTemplate(WithCatalogFile(bytes.NewReader(tc.catalog)))
			catalog, err := template.parseCatalogsSpec()
			tc.assertions(t, catalog, err)
		})
	}
}

var validComposite = `
schema: olm.composite
components:
  - name: first-catalog
    destination:
      path: my-operator
    strategy:
      name: semver
      template:
        schema: olm.builder.semver
        config:
          input: components/contribution1.yaml
          output: catalog.yaml
`

var invalidSchemaComposite = `
schema: invalid
components:
  - name: first-catalog
    destination:
      path: my-operator
    strategy:
      name: semver
      template:
        schema: olm.builder.semver
        config:
          input: components/contribution1.yaml
          output: catalog.yaml
`

func TestParseContributionSpec(t *testing.T) {
	type testCase struct {
		name       string
		composite  []byte
		assertions func(t *testing.T, composite *CompositeConfig, err error)
	}

	testCases := []testCase{
		{
			name:      "Valid composite",
			composite: []byte(validComposite),
			assertions: func(t *testing.T, composite *CompositeConfig, err error) {
				require.NoError(t, err)
				require.Equal(t, 1, len(composite.Components))
			},
		},
		{
			name:      "Unmarshal failure",
			composite: []byte(unmarshalFail),
			assertions: func(t *testing.T, composite *CompositeConfig, err error) {
				require.Error(t, err)
				require.Equal(t, "unmarshalling composite config: json: cannot unmarshal string into Go value of type composite.CompositeConfig", err.Error())
			},
		},
		{
			name:      "Invalid schema",
			composite: []byte(invalidSchemaComposite),
			assertions: func(t *testing.T, composite *CompositeConfig, err error) {
				require.Error(t, err)
				require.Equal(t, fmt.Sprintf("composite configuration file has unknown schema, should be %q", CompositeSchema), err.Error())
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			template := NewTemplate(WithContributionFile(bytes.NewReader(tc.composite)))
			contrib, err := template.parseContributionSpec()
			tc.assertions(t, contrib, err)
		})
	}
}

func TestNewCatalogBuilderMap(t *testing.T) {
	type testCase struct {
		name       string
		catalogs   []Catalog
		assertions func(t *testing.T, builderMap *CatalogBuilderMap, err error)
	}

	testCases := []testCase{
		{
			name: "Valid Catalogs",
			catalogs: []Catalog{
				{
					Name: "test-catalog",
					Destination: CatalogDestination{
						WorkingDir: "/",
						BaseImage:  "base",
					},
					Builders: []string{
						BasicBuilderSchema,
					},
				},
			},
			assertions: func(t *testing.T, builderMap *CatalogBuilderMap, err error) {
				require.NoError(t, err)
				//TODO: More assertions here
			},
		},
		{
			name: "Invalid Builder",
			catalogs: []Catalog{
				{
					Name: "test-catalog",
					Destination: CatalogDestination{
						WorkingDir: "/",
						BaseImage:  "base",
					},
					Builders: []string{
						"invalid",
					},
				},
			},
			assertions: func(t *testing.T, builderMap *CatalogBuilderMap, err error) {
				require.Error(t, err)
				require.Equal(t, "getting builder \"invalid\" for catalog \"test-catalog\": unknown schema \"invalid\"", err.Error())
			},
		},
		{
			name: "BaseImage+WorkingDir invalid",
			catalogs: []Catalog{
				{
					Name:        "test-catalog",
					Destination: CatalogDestination{},
					Builders: []string{
						BasicBuilderSchema,
					},
				},
			},
			assertions: func(t *testing.T, builderMap *CatalogBuilderMap, err error) {
				require.Error(t, err)
				require.Equal(t, "catalog configuration file field validation failed: \nCatalog test-catalog:\n  - destination.baseImage must not be an empty string\n  - destination.workingDir must not be an empty string\n", err.Error())
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			template := NewTemplate()
			builderMap, err := template.newCatalogBuilderMap(tc.catalogs, "yaml")
			tc.assertions(t, builderMap, err)
		})
	}
}

type fakeGetter struct {
	catalog     []byte
	shouldError bool
}

func (fg *fakeGetter) Get(url string) (*http.Response, error) {
	if fg.shouldError {
		return nil, fmt.Errorf("error!")
	}

	return &http.Response{
		Body: io.NopCloser(bytes.NewReader(fg.catalog)),
	}, nil
}

func TestFetchCatalogConfig(t *testing.T) {
	type testCase struct {
		name       string
		fakeGetter *fakeGetter
		path       string
		createFile bool
		assertions func(t *testing.T, rc io.ReadCloser, err error)
	}

	testCases := []testCase{
		{
			name: "Successful HTTP fetch",
			path: "http://some-path.com",
			fakeGetter: &fakeGetter{
				catalog: []byte(validCatalog),
			},
			assertions: func(t *testing.T, rc io.ReadCloser, err error) {
				require.NoError(t, err)
				require.NotNil(t, rc)
			},
		},
		{
			name: "Failed HTTP fetch",
			path: "http://some-path.com",
			fakeGetter: &fakeGetter{
				catalog:     []byte(validCatalog),
				shouldError: true,
			},
			assertions: func(t *testing.T, rc io.ReadCloser, err error) {
				require.Error(t, err)
				require.Equal(t, "fetching remote catalog config file \"http://some-path.com\": error!", err.Error())
			},
		},
		// TODO: for some reason this is triggering the fakeGetter.Get() function instead of using os.Open()
		{
			name: "Successful file fetch",
			path: "file/test.yaml",
			fakeGetter: &fakeGetter{
				catalog: []byte(validCatalog),
			},
			createFile: true,
			assertions: func(t *testing.T, rc io.ReadCloser, err error) {
				require.NoError(t, err)
				require.NotNil(t, rc)
			},
		},
		{
			name: "Failed file fetch",
			path: "file/test.yaml",
			fakeGetter: &fakeGetter{
				catalog: []byte(validCatalog),
			},
			createFile: false,
			assertions: func(t *testing.T, rc io.ReadCloser, err error) {
				require.Error(t, err)
				require.Equal(t, "opening catalog config file \"file/test.yaml\": open file/test.yaml: no such file or directory", err.Error())
			},
		},
	}

	testDir := t.TempDir()

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			filepath := tc.path
			if tc.createFile {
				err := os.MkdirAll(path.Join(testDir, path.Dir(tc.path)), 0o777)
				require.NoError(t, err)
				file, err := os.Create(path.Join(testDir, tc.path))
				require.NoError(t, err)
				_, err = file.WriteString(string(tc.fakeGetter.catalog))
				require.NoError(t, err)

				filepath = path.Join(testDir, tc.path)
			}

			rc, err := FetchCatalogConfig(filepath, tc.fakeGetter)
			tc.assertions(t, rc, err)
		})
	}
}
