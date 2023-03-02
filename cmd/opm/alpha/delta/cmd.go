package delta

import (
	"fmt"

	"github.com/blang/semver"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/operator-framework/operator-registry/alpha/action"
	"github.com/operator-framework/operator-registry/alpha/declcfg"
	"github.com/operator-framework/operator-registry/alpha/property"
	"github.com/operator-framework/operator-registry/cmd/opm/internal/util"
)

const humanReadabilityOnlyNote = `NOTE: This is meant to be used for convenience and human-readability only. The
CLI and output format are subject to change, so it is not recommended to depend
on the output in any programs or scripts. Use the "render" subcommand to do
more complex processing and automation.`

func NewDeltaCmd() *cobra.Command {
	logger := logrus.New()

	return &cobra.Command{
		Use:   "delta renderable1 renderable2",
		Short: "Identify unique members in each of two indices",
		Long: `The delta subcommand prints the bundles unique to each index.

` + humanReadabilityOnlyNote,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			reg, err := util.CreateCLIRegistry(cmd)
			if err != nil {
				logger.Fatal(err)
			}
			defer reg.Destroy()

			r1 := action.Render{
				Registry:       reg,
				AllowedRefMask: action.RefBundleImage | action.RefDCDir | action.RefDCImage | action.RefSqliteFile | action.RefSqliteImage,
				Refs:           args[0:],
			}

			r2 := action.Render{
				Registry:       reg,
				AllowedRefMask: action.RefBundleImage | action.RefDCDir | action.RefDCImage | action.RefSqliteFile | action.RefSqliteImage,
				Refs:           args[:1],
			}

			d1, err := r1.Run(cmd.Context())
			if err != nil {
				logger.Fatal(err)
			}
			d2, err := r2.Run(cmd.Context())
			if err != nil {
				logger.Fatal(err)
			}

			bv1, err := getBundleVersions(d1)
			if err != nil {
				logger.Fatal(err)
			}
			// fmt.Printf("%v\n", bv1)

			bv2, err := getBundleVersions(d2)
			if err != nil {
				logger.Fatal(err)
			}
			// fmt.Printf("%v\n", bv1)

			right := getUnique(bv1, bv2)
			left := getUnique(bv2, bv1)

			logger.Infof("unique entries for file %q", args[0])
			for name, version := range left {
				logger.Infof("--> %v / %v", name, version.String())
			}

			logger.Infof("unique entries for file %q", args[1])
			for name, version := range right {
				logger.Infof("--> %v / %v", name, version.String())
			}

			return nil
		},
	}
}

func getUnique(m1, m2 map[string]semver.Version) map[string]semver.Version {
	unique := make(map[string]semver.Version)

	for name1, ver1 := range m1 {
		if _, ok := m2[name1]; !ok {
			unique[name1] = ver1
		} else {
			if !m2[name1].EQ(ver1) {
				unique[name1] = ver1
			}
		}
	}
	return unique
}

func getBundleVersions(d *declcfg.DeclarativeConfig) (map[string]semver.Version, error) {
	versions := make(map[string]semver.Version)

	for _, b := range d.Bundles {
		props, err := property.Parse(b.Properties)
		if err != nil {
			return nil, fmt.Errorf("parse properties for bundle %q: %v", b.Name, err)
		}
		if len(props.Packages) != 1 {
			return nil, fmt.Errorf("bundle %q has multiple %q properties, expected exactly 1", b.Name, property.TypePackage)
		}
		v, err := semver.Parse(props.Packages[0].Version)
		if err != nil {
			return nil, fmt.Errorf("bundle %q has invalid version %q: %v", b.Name, props.Packages[0].Version, err)
		}

		versions[b.Name] = v
	}

	return versions, nil
}
