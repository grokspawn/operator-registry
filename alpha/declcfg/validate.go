package declcfg

import (
	"fmt"

	"github.com/blang/semver/v4"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/validation"

	"github.com/operator-framework/operator-registry/alpha/property"
)

// Validate validates a DeclarativeConfig without converting to model.
// It performs the same validation checks as ConvertToModel but doesn't build the model structure.
func Validate(cfg DeclarativeConfig) error {
	packageNames := sets.New[string]()
	defaultChannels := map[string]string{}

	// Validate packages
	for _, p := range cfg.Packages {
		if p.Name == "" {
			return fmt.Errorf("config contains package with no name")
		}

		if packageNames.Has(p.Name) {
			return fmt.Errorf("duplicate package %q", p.Name)
		}
		packageNames.Insert(p.Name)

		if errs := validation.IsDNS1123Label(p.Name); len(errs) > 0 {
			return fmt.Errorf("invalid package name %q: %v", p.Name, errs)
		}

		defaultChannels[p.Name] = p.DefaultChannel
	}

	// Validate channels
	packageChannels := make(map[string]sets.Set[string])
	channelDefinedEntries := map[string]sets.Set[string]{}
	for _, c := range cfg.Channels {
		if !packageNames.Has(c.Package) {
			return fmt.Errorf("unknown package %q for channel %q", c.Package, c.Name)
		}

		if c.Name == "" {
			return fmt.Errorf("package %q contains channel with no name", c.Package)
		}

		if _, ok := packageChannels[c.Package]; !ok {
			packageChannels[c.Package] = sets.New[string]()
		}
		if packageChannels[c.Package].Has(c.Name) {
			return fmt.Errorf("package %q has duplicate channel %q", c.Package, c.Name)
		}
		packageChannels[c.Package].Insert(c.Name)

		// Track entries defined in channel
		cde := sets.Set[string]{}
		seenEntries := sets.New[string]()
		for _, entry := range c.Entries {
			if seenEntries.Has(entry.Name) {
				return fmt.Errorf("invalid package %q, channel %q: duplicate entry %q", c.Package, c.Name, entry.Name)
			}
			seenEntries.Insert(entry.Name)
			cde = cde.Insert(entry.Name)
		}
		channelDefinedEntries[c.Package] = cde
	}

	// Validate bundles
	packageBundles := map[string]sets.Set[string]{}
	for _, b := range cfg.Bundles {
		if b.Package == "" {
			return fmt.Errorf("package name must be set for bundle %q", b.Name)
		}
		if !packageNames.Has(b.Package) {
			return fmt.Errorf("unknown package %q for bundle %q", b.Package, b.Name)
		}

		bundles, ok := packageBundles[b.Package]
		if !ok {
			bundles = sets.Set[string]{}
		}
		if bundles.Has(b.Name) {
			return fmt.Errorf("package %q has duplicate bundle %q", b.Package, b.Name)
		}
		bundles.Insert(b.Name)
		packageBundles[b.Package] = bundles

		props, err := property.Parse(b.Properties)
		if err != nil {
			return fmt.Errorf("parse properties for bundle %q: %v", b.Name, err)
		}

		if len(props.Packages) != 1 {
			return fmt.Errorf("package %q bundle %q must have exactly 1 %q property, found %d", b.Package, b.Name, property.TypePackage, len(props.Packages))
		}

		if b.Package != props.Packages[0].PackageName {
			return fmt.Errorf("package %q does not match %q property %q", b.Package, property.TypePackage, props.Packages[0].PackageName)
		}

		if err := validateImagePullSpec(b.Image, "package %q bundle %q image", b.Package, b.Name); err != nil {
			return err
		}
		for i, rel := range b.RelatedImages {
			if err := validateImagePullSpec(rel.Image, "package %q bundle %q relatedImages[%d].image", b.Package, b.Name, i); err != nil {
				return err
			}
		}

		// Validate version
		rawVersion := props.Packages[0].Version
		if _, err := semver.Parse(rawVersion); err != nil {
			return fmt.Errorf("error parsing bundle %q version %q: %v", b.Name, rawVersion, err)
		}

		channelDefinedEntries[b.Package] = channelDefinedEntries[b.Package].Delete(b.Name)

		// Check that bundle is in at least one channel
		found := false
		for _, ch := range cfg.Channels {
			if ch.Package != b.Package {
				continue
			}
			for _, entry := range ch.Entries {
				if entry.Name == b.Name {
					found = true
					break
				}
			}
			if found {
				break
			}
		}
		if !found {
			return fmt.Errorf("package %q, bundle %q not found in any channel entries", b.Package, b.Name)
		}
	}

	// Check for channel entries without bundles
	for pkg, entries := range channelDefinedEntries {
		if entries.Len() > 0 {
			return fmt.Errorf("no olm.bundle blobs found in package %q for olm.channel entries %s", pkg, sets.List[string](entries))
		}
	}

	// Validate default channels exist
	for pkg, defaultChannel := range defaultChannels {
		if defaultChannel != "" && !packageChannels[pkg].Has(defaultChannel) {
			return fmt.Errorf("package %q references non-existent default channel %q", pkg, defaultChannel)
		}
	}

	// Validate deprecations
	deprecationsByPackage := sets.New[string]()
	for i, deprecation := range cfg.Deprecations {
		if deprecation.Package == "" {
			return fmt.Errorf("package name must be set for deprecation item %v", i)
		}

		if !packageNames.Has(deprecation.Package) {
			return fmt.Errorf("cannot apply deprecations to an unknown package %q", deprecation.Package)
		}

		if deprecationsByPackage.Has(deprecation.Package) {
			return fmt.Errorf("expected a maximum of one deprecation per package: %q", deprecation.Package)
		}
		deprecationsByPackage.Insert(deprecation.Package)

		references := sets.New[PackageScopedReference]()
		for j, entry := range deprecation.Entries {
			if entry.Reference.Schema == "" {
				return fmt.Errorf("schema must be set for deprecation entry [%v] for package %q", j, deprecation.Package)
			}

			if references.Has(entry.Reference) {
				return fmt.Errorf("duplicate deprecation entry %#v for package %q", entry.Reference, deprecation.Package)
			}
			references.Insert(entry.Reference)

			switch entry.Reference.Schema {
			case SchemaBundle:
				if !packageBundles[deprecation.Package].Has(entry.Reference.Name) {
					return fmt.Errorf("cannot deprecate bundle %q for package %q: bundle not found", entry.Reference.Name, deprecation.Package)
				}
			case SchemaChannel:
				if !packageChannels[deprecation.Package].Has(entry.Reference.Name) {
					return fmt.Errorf("cannot deprecate channel %q for package %q: channel not found", entry.Reference.Name, deprecation.Package)
				}
			case SchemaPackage:
				if entry.Reference.Name != "" {
					return fmt.Errorf("package name must be empty for deprecated package %q (specified %q)", deprecation.Package, entry.Reference.Name)
				}
			default:
				return fmt.Errorf("cannot deprecate object %#v referenced by entry %v for package %q: object schema unknown", entry.Reference, j, deprecation.Package)
			}
		}
	}

	// TODO: Validate channel graphs (no circular replaces, valid heads, etc.)
	// This would require building the graph structure, which is what ConvertToModel does.
	// For now, this validation is "good enough" and catches most common errors.

	return nil
}
