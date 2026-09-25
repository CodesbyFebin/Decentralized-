// Package manifest parses, validates, normalizes and hashes dh/v1
// Application manifests.
//
// Operators write YAML with friendly quantities ("500m", "512Mi", "10s").
// Parse turns that into an api.Manifest with integer quantities and every
// default filled in, so the normalized form — and therefore its hash — does
// not depend on how the operator happened to spell it.
package manifest

import (
	"bytes"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"go.yaml.in/yaml/v3"

	"decentralized.host/pkg/api"
	"decentralized.host/pkg/envelope"
)

var (
	nameRE    = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{0,40}[a-z0-9])?$`)
	b3Image   = regexp.MustCompile(`^([a-z0-9][a-z0-9._-]{0,63})@b3:([a-f0-9]{64})$`)
	dockerRef = regexp.MustCompile(`^[a-z0-9][a-z0-9._/:-]*@sha256:([a-f0-9]{64})$`)
	hostRE    = regexp.MustCompile(`^(\*\.)?([a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?\.)+[a-z0-9-]{2,63}$`)
	envKeyRE  = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]{0,127}$`)
)

// raw mirrors the YAML the operator writes.
type raw struct {
	APIVersion string `yaml:"apiVersion"`
	Kind       string `yaml:"kind"`
	Metadata   struct {
		Name  string `yaml:"name"`
		Owner string `yaml:"owner"`
	} `yaml:"metadata"`
	Spec struct {
		Replicas  *int64            `yaml:"replicas"`
		Runtime   string            `yaml:"runtime"`
		Image     string            `yaml:"image"`
		Command   []string          `yaml:"command"`
		Args      []string          `yaml:"args"`
		Env       map[string]string `yaml:"env"`
		Resources struct {
			CPU      string `yaml:"cpu"`
			Mem      string `yaml:"mem"`
			Memory   string `yaml:"memory"`
			Requests struct {
				CPU    string `yaml:"cpu"`
				Memory string `yaml:"memory"`
			} `yaml:"requests"`
		} `yaml:"resources"`
		Placement struct {
			Tiers        []string `yaml:"tiers"`
			Spread       string   `yaml:"spread"`
			AntiAffinity string   `yaml:"antiAffinity"`
			Arch         []string `yaml:"arch"`
			Features     []string `yaml:"features"`
			PreferRegion string   `yaml:"preferRegion"`
			Federation   []string `yaml:"federation"`
			Require      struct {
				Arch     []string `yaml:"arch"`
				Features []string `yaml:"features"`
			} `yaml:"require"`
		} `yaml:"placement"`
		Ports []struct {
			Name      string `yaml:"name"`
			Container int64  `yaml:"container"`
			Protocol  string `yaml:"protocol"`
		} `yaml:"ports"`
		Volumes []struct {
			Name       string `yaml:"name"`
			Size       string `yaml:"size"`
			Mount      string `yaml:"mount"`
			Durability struct {
				Replicas *int64 `yaml:"replicas"`
				Erasure  string `yaml:"erasure"`
			} `yaml:"durability"`
			Snapshot struct {
				Every  string `yaml:"every"`
				Retain int64  `yaml:"retain"`
			} `yaml:"snapshot"`
		} `yaml:"volumes"`
		Ingress []struct {
			Host   string `yaml:"host"`
			Port   string `yaml:"port"`
			TLS    string `yaml:"tls"`
			Issuer string `yaml:"issuer"`
		} `yaml:"ingress"`
		Health struct {
			HTTP             string `yaml:"http"`
			Interval         string `yaml:"interval"`
			Timeout          string `yaml:"timeout"`
			FailureThreshold int64  `yaml:"failureThreshold"`
		} `yaml:"health"`
		Update struct {
			Strategy       string `yaml:"strategy"`
			MaxUnavailable *int64 `yaml:"maxUnavailable"`
		} `yaml:"update"`
	} `yaml:"spec"`
}

// Errors collects every validation problem, not just the first.
type Errors []string

func (e Errors) Error() string { return "manifest: " + strings.Join(e, "; ") }

// Parse reads one YAML (or JSON) manifest.
func Parse(src []byte) (*api.Manifest, error) {
	var r raw
	dec := yaml.NewDecoder(bytes.NewReader(src))
	dec.KnownFields(true)
	if err := dec.Decode(&r); err != nil {
		return nil, Errors{"yaml: " + err.Error()}
	}
	return normalize(&r)
}

func normalize(r *raw) (*api.Manifest, error) {
	var errs Errors
	add := func(format string, args ...any) { errs = append(errs, fmt.Sprintf(format, args...)) }

	if r.APIVersion != api.ProtocolVersion {
		add("apiVersion must be %s, got %q", api.ProtocolVersion, r.APIVersion)
	}
	if r.Kind != "Application" {
		add("kind must be Application, got %q", r.Kind)
	}
	m := &api.Manifest{APIVersion: api.ProtocolVersion, Kind: "Application"}
	m.Metadata.Name = r.Metadata.Name
	m.Metadata.Owner = r.Metadata.Owner
	if !nameRE.MatchString(m.Metadata.Name) {
		add("metadata.name %q must be 1-42 lowercase letters, digits or dashes", m.Metadata.Name)
	}

	s := &m.Spec
	s.Replicas = 1
	if r.Spec.Replicas != nil {
		s.Replicas = *r.Spec.Replicas
	}
	if s.Replicas < 0 || s.Replicas > 64 {
		add("spec.replicas must be 0..64")
	}

	s.Image = strings.TrimSpace(r.Spec.Image)
	s.Runtime = r.Spec.Runtime
	switch {
	case s.Image == "":
		add("spec.image is required")
	case b3Image.MatchString(s.Image):
		if s.Runtime == "" {
			s.Runtime = "process"
		}
	case dockerRef.MatchString(s.Image):
		if s.Runtime == "" {
			s.Runtime = "docker"
		}
	default:
		add("spec.image %q must be digest-pinned: name@b3:<64 hex> (process artifact) or ref@sha256:<64 hex> (container)", s.Image)
	}
	switch s.Runtime {
	case "process":
		if s.Image != "" && !b3Image.MatchString(s.Image) {
			add("runtime process needs an artifact image name@b3:<hex>")
		}
	case "docker":
		if s.Image != "" && !dockerRef.MatchString(s.Image) {
			add("runtime docker needs an image ref@sha256:<hex>")
		}
	case "":
	default:
		add("spec.runtime %q is not supported (process, docker)", s.Runtime)
	}

	s.Command = append([]string{}, r.Spec.Command...)
	s.Command = append(s.Command, r.Spec.Args...)
	if s.Command == nil {
		s.Command = []string{}
	}
	s.Env = map[string]string{}
	for k, v := range r.Spec.Env {
		if !envKeyRE.MatchString(k) || strings.HasPrefix(k, "DH_") {
			add("spec.env key %q is invalid or reserved (DH_ prefix)", k)
			continue
		}
		s.Env[k] = v
	}

	cpuSrc := first(r.Spec.Resources.CPU, r.Spec.Resources.Requests.CPU, "100m")
	memSrc := first(r.Spec.Resources.Mem, r.Spec.Resources.Memory, r.Spec.Resources.Requests.Memory, "64Mi")
	if v, err := ParseCPU(cpuSrc); err != nil {
		add("spec.resources.cpu: %v", err)
	} else {
		s.Resources.CPUMilli = v
	}
	if v, err := ParseBytes(memSrc); err != nil {
		add("spec.resources.mem: %v", err)
	} else {
		s.Resources.MemBytes = v
	}

	p := &s.Placement
	p.Tiers = sortedSet(r.Spec.Placement.Tiers)
	if len(p.Tiers) == 0 {
		p.Tiers = []string{"trusted"}
	}
	p.Spread = first(r.Spec.Placement.Spread, "failure-domain")
	if p.Spread != "failure-domain" && p.Spread != "none" {
		add("spec.placement.spread must be failure-domain or none")
	}
	p.AntiAffinity = first(r.Spec.Placement.AntiAffinity, "hard")
	if p.AntiAffinity != "hard" && p.AntiAffinity != "soft" && p.AntiAffinity != "none" {
		add("spec.placement.antiAffinity must be hard, soft or none")
	}
	p.Arch = sortedSet(append(r.Spec.Placement.Arch, r.Spec.Placement.Require.Arch...))
	p.Features = sortedSet(append(r.Spec.Placement.Features, r.Spec.Placement.Require.Features...))
	p.PreferRegion = r.Spec.Placement.PreferRegion
	p.Federation = sortedSet(r.Spec.Placement.Federation)

	s.Ports = []api.Port{}
	seenPort := map[string]bool{}
	for i, rp := range r.Spec.Ports {
		name := first(rp.Name, "http")
		if !nameRE.MatchString(name) || seenPort[name] {
			add("spec.ports[%d].name %q is invalid or duplicated", i, name)
		}
		seenPort[name] = true
		proto := first(rp.Protocol, "http")
		if proto != "http" && proto != "tcp" {
			add("spec.ports[%d].protocol must be http or tcp", i)
		}
		if rp.Container < 0 || rp.Container > 65535 {
			add("spec.ports[%d].container out of range", i)
		}
		s.Ports = append(s.Ports, api.Port{Name: name, Container: rp.Container, Protocol: proto})
	}
	if len(s.Ports) > 1 {
		add("spec.ports: this release exposes at most one port per workload")
	}

	s.Volumes = []api.VolumeSpec{}
	seenVol := map[string]bool{}
	for i, rv := range r.Spec.Volumes {
		v := api.VolumeSpec{Name: rv.Name, Mount: rv.Mount}
		if !nameRE.MatchString(v.Name) || seenVol[v.Name] {
			add("spec.volumes[%d].name %q is invalid or duplicated", i, v.Name)
		}
		seenVol[v.Name] = true
		size, err := ParseBytes(first(rv.Size, "1Gi"))
		if err != nil {
			add("spec.volumes[%d].size: %v", i, err)
		}
		v.SizeBytes = size
		if v.Mount == "" {
			v.Mount = "/data/" + v.Name
		}
		v.Durability.Replicas = 2
		if rv.Durability.Replicas != nil {
			v.Durability.Replicas = *rv.Durability.Replicas
		}
		if v.Durability.Replicas < 1 || v.Durability.Replicas > 7 {
			add("spec.volumes[%d].durability.replicas must be 1..7", i)
		}
		v.Durability.Erasure = first(rv.Durability.Erasure, "none")
		if v.Durability.Erasure != "none" {
			add("spec.volumes[%d].durability.erasure %q: erasure coding is not implemented; use none", i, v.Durability.Erasure)
		}
		every, err := ParseDuration(first(rv.Snapshot.Every, "30s"))
		if err != nil || every < 1000 {
			add("spec.volumes[%d].snapshot.every must be a duration >= 1s", i)
		}
		v.SnapshotEveryMs = every
		v.RetainSnapshots = rv.Snapshot.Retain
		if v.RetainSnapshots == 0 {
			v.RetainSnapshots = 5
		}
		s.Volumes = append(s.Volumes, v)
	}

	s.Ingress = []api.Ingress{}
	for i, ri := range r.Spec.Ingress {
		in := api.Ingress{Host: strings.ToLower(ri.Host), Port: first(ri.Port, "http"), TLS: first(ri.TLS, "none"), Issuer: ri.Issuer}
		if !hostRE.MatchString(in.Host) {
			add("spec.ingress[%d].host %q is not a valid DNS name", i, ri.Host)
		}
		if !seenPort[in.Port] {
			add("spec.ingress[%d].port %q does not name a declared port", i, in.Port)
		}
		if in.TLS != "acme" && in.TLS != "local" && in.TLS != "none" {
			add("spec.ingress[%d].tls must be acme, local or none", i)
		}
		s.Ingress = append(s.Ingress, in)
	}

	s.Health.HTTP = r.Spec.Health.HTTP
	if s.Health.HTTP != "" && !strings.HasPrefix(s.Health.HTTP, "/") {
		add("spec.health.http must be a path starting with /")
	}
	if iv, err := ParseDuration(first(r.Spec.Health.Interval, "5s")); err != nil || iv < 500 {
		add("spec.health.interval must be a duration >= 500ms")
	} else {
		s.Health.IntervalMs = iv
	}
	if to, err := ParseDuration(first(r.Spec.Health.Timeout, "1s")); err != nil || to < 50 {
		add("spec.health.timeout must be a duration >= 50ms")
	} else {
		s.Health.TimeoutMs = to
	}
	s.Health.FailureThreshold = r.Spec.Health.FailureThreshold
	if s.Health.FailureThreshold == 0 {
		s.Health.FailureThreshold = 3
	}

	s.Update.Strategy = first(r.Spec.Update.Strategy, "rolling")
	if s.Update.Strategy != "rolling" {
		add("spec.update.strategy must be rolling")
	}
	s.Update.MaxUnavailable = 1
	if r.Spec.Update.MaxUnavailable != nil {
		s.Update.MaxUnavailable = *r.Spec.Update.MaxUnavailable
	}
	if s.Update.MaxUnavailable < 1 {
		add("spec.update.maxUnavailable must be >= 1")
	}

	if len(errs) > 0 {
		return nil, errs
	}
	return m, nil
}

// Hash is the content address of a normalized manifest.
func Hash(m *api.Manifest) string { return envelope.DomainHash("manifest", m) }

// Digest returns the digest part of an image reference ("b3:<hex>" or "sha256:<hex>").
func Digest(image string) string {
	if i := strings.LastIndex(image, "@"); i >= 0 {
		return image[i+1:]
	}
	return ""
}

// ArtifactName returns the name part of a process artifact image.
func ArtifactName(image string) string {
	if m := b3Image.FindStringSubmatch(image); m != nil {
		return m[1]
	}
	return ""
}

// ParseCPU parses "500m", "2", "0.5" into millicores.
func ParseCPU(s string) (int64, error) {
	s = strings.TrimSpace(s)
	if strings.HasSuffix(s, "m") {
		n, err := strconv.ParseInt(strings.TrimSuffix(s, "m"), 10, 64)
		if err != nil || n < 0 {
			return 0, fmt.Errorf("invalid cpu %q", s)
		}
		return n, nil
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil || f < 0 {
		return 0, fmt.Errorf("invalid cpu %q", s)
	}
	return int64(f*1000 + 0.5), nil
}

// ParseBytes parses "512Mi", "10Gi", "1G", "4096" into bytes.
func ParseBytes(s string) (int64, error) {
	s = strings.TrimSpace(s)
	units := []struct {
		suffix string
		mult   int64
	}{{"Ki", 1 << 10}, {"Mi", 1 << 20}, {"Gi", 1 << 30}, {"Ti", 1 << 40}, {"K", 1000}, {"M", 1000 * 1000}, {"G", 1000 * 1000 * 1000}, {"T", 1000 * 1000 * 1000 * 1000}}
	for _, u := range units {
		if strings.HasSuffix(s, u.suffix) {
			f, err := strconv.ParseFloat(strings.TrimSuffix(s, u.suffix), 64)
			if err != nil || f < 0 {
				return 0, fmt.Errorf("invalid quantity %q", s)
			}
			return int64(f * float64(u.mult)), nil
		}
	}
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil || n < 0 {
		return 0, fmt.Errorf("invalid quantity %q", s)
	}
	return n, nil
}

// ParseDuration parses a Go duration into milliseconds.
func ParseDuration(s string) (int64, error) {
	d, err := time.ParseDuration(strings.TrimSpace(s))
	if err != nil {
		return 0, err
	}
	return d.Milliseconds(), nil
}

// FormatBytes renders a byte count with binary units.
func FormatBytes(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%dB", n)
	}
	div, exp := int64(unit), 0
	for m := n / unit; m >= unit; m /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f%ciB", float64(n)/float64(div), "KMGTPE"[exp])
}

func first(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func sortedSet(in []string) []string {
	set := map[string]bool{}
	for _, s := range in {
		if s = strings.TrimSpace(s); s != "" {
			set[s] = true
		}
	}
	out := make([]string, 0, len(set))
	for s := range set {
		out = append(out, s)
	}
	sort.Strings(out)
	return out
}
