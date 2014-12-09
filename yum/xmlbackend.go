package yum

import (
	"compress/gzip"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"

	"github.com/gonuts/logger"
)

// RepositoryXMLBackend is a Backend querying YUM XML repositories
type RepositoryXMLBackend struct {
	Name       string
	Packages   map[string][]*Package
	Provides   map[string][]*Provides
	DBName     string
	Primary    string
	Repository *Repository
	msg        *logger.Logger
}

func NewRepositoryXMLBackend(repo *Repository) (*RepositoryXMLBackend, error) {
	const dbname = "primary.xml.gz"
	return &RepositoryXMLBackend{
		Name:       "RepositoryXMLBackend",
		Packages:   make(map[string][]*Package),
		Provides:   make(map[string][]*Provides),
		DBName:     dbname,
		Primary:    filepath.Join(repo.CacheDir, dbname),
		Repository: repo,
		msg:        repo.msg,
	}, nil
}

// Close cleans up a backend after use
func (repo *RepositoryXMLBackend) Close() error {
	return nil
}

// YumDataType returns the ID for the data type as used in the repomd.xml file
func (repo *RepositoryXMLBackend) YumDataType() string {
	return "primary"
}

// Download the DB from server
func (repo *RepositoryXMLBackend) GetLatestDB(url string) error {
	var err error
	out, err := os.Create(repo.Primary)
	if err != nil {
		return err
	}
	defer out.Close()

	r, err := getRemoteData(url)
	if err != nil {
		return err
	}
	defer r.Close()
	_, err = io.Copy(out, r)
	return err
}

// Check whether the DB is there
func (repo *RepositoryXMLBackend) HasDB() bool {
	return path_exists(repo.Primary)
}

// Load loads the DB
func (repo *RepositoryXMLBackend) LoadDB() error {
	var err error

	repo.msg.Debugf("start parsing metadata XML file... (%s)\n", repo.Primary)
	type xmlTree struct {
		XMLName  xml.Name `xml:"metadata"`
		Packages []struct {
			Type string `xml:"type,attr"`
			Name string `xml:"name"`
			Arch string `xml:"arch"`

			Version struct {
				Epoch   string `xml:"epoch,attr"`
				Version string `xml:"ver,attr"`
				Release string `xml:"rel,attr"`
			} `xml:"version"`

			Checksum struct {
				Value string `xml:",innerxml"`
				Type  string `xml:"type,attr"`
				PkgId string `xml:"pkgid,attr"`
			} `xml:"checksum"`

			Summary  string `xml:"summary"`
			Descr    string `xml:"description"`
			Packager string `xml:"packager"`
			Url      string `xml:"url"`

			Time struct {
				File  string `xml:"file,attr"`
				Build string `xml:"build,attr"`
			} `xml:"time"`

			Size struct {
				Package   int64 `xml:"package,attr"`
				Installed int64 `xml:"installed,attr"`
				Archive   int64 `xml:"archive,attr"`
			} `xml:"size"`

			Location struct {
				Href string `xml:"href,attr"`
			} `xml:"location"`

			Format struct {
				License   string `xml:"license"`
				Vendor    string `xml:"vendor"`
				Group     string `xml:"group"`
				BuildHost string `xml:"buildhost"`
				SourceRpm string `xml:"sourcerpm"`

				HeaderRange struct {
					Beg int64 `xml:"start,attr"`
					End int64 `xml:"end,attr"`
				} `xml:"header-range"`

				Provides []struct {
					Name    string `xml:"name,attr"`
					Flags   string `xml:"flags,attr"`
					Epoch   string `xml:"epoch,attr"`
					Version string `xml:"ver,attr"`
					Release string `xml:"rel,attr"`
				} `xml:"provides>entry"`

				Requires []struct {
					Name    string `xml:"name,attr"`
					Flags   string `xml:"flags,attr"`
					Epoch   string `xml:"epoch,attr"`
					Version string `xml:"ver,attr"`
					Release string `xml:"rel,attr"`
					Pre     string `xml:"pre,attr"`
				} `xml:"requires>entry"`

				Files []string `xml:"file"`
			} `xml:"format"`
		} `xml:"package"`
	}

	// load the yum XML package list
	f, err := os.Open(repo.Primary)
	if err != nil {
		return err
	}
	defer f.Close()

	var r io.Reader
	if rr, err := gzip.NewReader(f); err != nil {
		if err != gzip.ErrHeader {
			return err
		}
		// perhaps not a compressed file after all...
		_, err = f.Seek(0, 0)
		if err != nil {
			return err
		}
		r = f
	} else {
		r = rr
		defer rr.Close()
	}

	var tree xmlTree
	err = xml.NewDecoder(r).Decode(&tree)
	if err != nil {
		return err
	}

	for _, xml := range tree.Packages {
		pkg := NewPackage(
			xml.Name, xml.Version.Version, xml.Version.Release,
			xml.Version.Epoch,
		)
		pkg.arch = xml.Arch
		pkg.group = xml.Format.Group
		pkg.location = xml.Location.Href
		for _, v := range xml.Format.Provides {
			prov := NewProvides(
				v.Name,
				v.Version,
				v.Release,
				v.Epoch,
				v.Flags,
				pkg,
			)
			pkg.provides = append(pkg.provides, prov)

			if !str_in_slice(prov.Name(), IGNORED_PACKAGES) {
				repo.Provides[prov.Name()] = append(repo.Provides[prov.Name()], prov)
			}
		}

		for _, v := range xml.Format.Requires {
			req := NewRequires(
				v.Name,
				v.Version,
				v.Release,
				v.Epoch,
				v.Flags,
				v.Pre,
			)
			pkg.requires = append(pkg.requires, req)
		}
		pkg.repository = repo.Repository

		// add package to repository
		repo.Packages[pkg.Name()] = append(repo.Packages[pkg.Name()], pkg)
		repo.msg.Debugf(
			"(repo=%s) added package: %s.%s-%s\n",
			repo.Primary,
			pkg.Name(),
			pkg.Version(),
			pkg.Release(),
		)
	}

	repo.msg.Debugf("start parsing metadata XML file... (%s) [done]\n", repo.Primary)
	return err
}

// FindLatestMatchingName locats a package by name, returns the latest available version.
func (repo *RepositoryXMLBackend) FindLatestMatchingName(name, version, release string) (*Package, error) {
	var pkg *Package
	var err error

	pkgs, ok := repo.Packages[name]
	if !ok {
		repo.msg.Debugf("could not find package %q\n", name)
		return nil, fmt.Errorf("no such package %q", name)
	}

	if version == "" && len(pkgs) > 0 {
		// return latest
		sorted := make([]*Package, len(pkgs))
		copy(sorted, pkgs)
		sort.Sort(Packages(sorted))
		pkg = sorted[len(sorted)-1]
	} else {
		// trying to match the requirements
		req := NewRequires(name, version, release, "", "EQ", "")
		sorted := make(Packages, 0, len(pkgs))
		for _, p := range pkgs {
			if req.ProvideMatches(p) {
				sorted = append(sorted, p)
			}
		}
		if len(sorted) > 0 {
			sort.Sort(sorted)
			pkg = sorted[len(sorted)-1]
		}
	}

	return pkg, err
}

// FindLatestMatchingRequire locates a package providing a given functionality.
func (repo *RepositoryXMLBackend) FindLatestMatchingRequire(requirement *Requires) (*Package, error) {
	var pkg *Package
	var err error

	repo.msg.Debugf("looking for match for %v\n", requirement)

	pkgs, ok := repo.Provides[requirement.Name()]
	if !ok {
		repo.msg.Debugf("could not find package providing %s-%s\n", requirement.Name(), requirement.Version())
		return nil, fmt.Errorf("no package providing name=%q version=%q release=%q",
			requirement.Name(), requirement.Version(), requirement.Release(),
		)
	}

	if requirement.Version() == "" && len(pkgs) > 0 {
		// return latest
		sorted := make([]RPM, 0, len(pkgs))
		for _, p := range pkgs {
			sorted = append(sorted, p)
		}
		sort.Sort(RPMSlice(sorted))
		pkg = sorted[len(sorted)-1].(*Provides).Package
	} else {
		// trying to match the requirements
		sorted := make(RPMSlice, 0, len(pkgs))
		for _, p := range pkgs {
			if requirement.ProvideMatches(p) {
				sorted = append(sorted, p)
			}
		}
		if len(sorted) > 0 {
			sort.Sort(sorted)
			pkg = sorted[len(sorted)-1].(*Provides).Package
			repo.msg.Debugf("found %d version matching - returning latest: %s.%s-%s\n", len(sorted), pkg.Name(), pkg.Version(), pkg.Release())
		}
	}

	return pkg, err
}

// GetPackages returns all the packages known by a YUM repository
func (repo *RepositoryXMLBackend) GetPackages() []*Package {
	pkgs := make([]*Package, 0, len(repo.Packages))
	for _, pkg := range repo.Packages {
		pkgs = append(pkgs, pkg...)
	}
	return pkgs
}

func init() {
	g_backends["RepositoryXMLBackend"] = func(repo *Repository) (Backend, error) {
		return NewRepositoryXMLBackend(repo)
	}
}
