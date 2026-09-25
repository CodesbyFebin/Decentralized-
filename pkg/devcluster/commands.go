package devcluster

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"text/tabwriter"
	"time"

	"decentralized.host/pkg/cli"
)

func init() {
	cli.Register("dev up", "run a real local multi-process cluster: --dir DIR [--cps 3] [--hosts 3] [--edges 1] [--base 17700] [--sample]", devUp)
	cli.Register("dev down", "stop a local cluster and its workloads: --dir DIR", devDown)
	cli.Register("dev status", "show processes of a local cluster: --dir DIR", devStatus)
}

// SampleManifest is the demo application (image filled in at run time).
const SampleManifest = `apiVersion: dh/v1
kind: Application
metadata:
  name: web
  owner: dev
spec:
  replicas: 3
  image: %s
  env:
    MESSAGE: "hello from a sovereign host"
  resources:
    cpu: 100m
    mem: 64Mi
  placement:
    tiers: [trusted]
    spread: failure-domain
    antiAffinity: hard
  ports:
    - name: http
  volumes:
    - name: data
      size: 256Mi
      mount: /data
      durability:
        replicas: 2
      snapshot:
        every: 5s
        retain: 3
  ingress:
    - host: web.dev.test
      port: http
      tls: local
  health:
    http: /healthz
    interval: 2s
`

func devUp(_ *cli.Operator, args []string) error {
	fs := flag.NewFlagSet("dev up", flag.ContinueOnError)
	var o Options
	fs.StringVar(&o.Dir, "dir", "./devcluster", "cluster directory")
	fs.IntVar(&o.CPs, "cps", 3, "control-plane members (1, 3 or 5)")
	fs.IntVar(&o.Hosts, "hosts", 3, "workload hosts")
	fs.IntVar(&o.Edges, "edges", 1, "edge hosts")
	fs.IntVar(&o.Base, "base", 17700, "base port")
	fs.StringVar(&o.Cluster, "cluster", "dev", "cluster name")
	fs.BoolVar(&o.Chaos, "chaos", true, "enable host fault-injection on the local status API")
	fs.StringVar(&o.Postgres, "postgres", "", "optional Postgres URL for the evidence mirror")
	fs.BoolVar(&o.TLS, "tls", false, "serve every member API over TLS (production shape; the console then needs the root CA trusted)")
	sample := fs.Bool("sample", true, "push dh-beacon and deploy the sample app")
	if err := fs.Parse(args); err != nil {
		return err
	}
	c, err := Up(o)
	if err != nil {
		return err
	}
	if *sample {
		fmt.Println("• pushing the dh-beacon artifact and applying the sample app")
		image, _, err := c.Op.PushArtifact(filepath.Join(c.Opt.Bin, "dh-beacon"), "dh-beacon", true)
		if err != nil {
			return err
		}
		m := fmt.Sprintf(SampleManifest, image)
		_ = os.WriteFile(filepath.Join(c.Opt.Dir, "web.yaml"), []byte(m), 0o644)
		r, err := c.Op.Apply([]byte(m))
		if err != nil {
			return err
		}
		fmt.Println(" ", r.Message)
	}
	tok := c.Op.Token(12*time.Hour, []string{"api.read", "api.write", "api.admin"}, "console")
	fmt.Printf(`
Cluster %q is running: %d control-plane member(s), %d host(s), %d edge.
  console:   http://%s/#token=%s
  operator:  export DH_HOME=%s
  edge:      curl -H 'Host: web.dev.test' -H 'X-DH-No-Redirect: 1' http://%s/
  logs:      %s
  stop:      dh dev down --dir %s
`, o.Cluster, o.CPs, o.Hosts+o.Edges, o.Edges, c.Procs[0].API, tok, c.Home, c.Edge, filepath.Join(c.Opt.Dir, "logs"), c.Opt.Dir)
	return nil
}

func devDown(_ *cli.Operator, args []string) error {
	fs := flag.NewFlagSet("dev down", flag.ContinueOnError)
	dir := fs.String("dir", "./devcluster", "cluster directory")
	if err := fs.Parse(args); err != nil {
		return err
	}
	c, err := Load(*dir)
	if err != nil {
		return err
	}
	procs, workloads := c.Down()
	fmt.Printf("stopped %d process(es) and %d workload(s) of %s\n", procs, workloads, c.Opt.Dir)
	return nil
}

func devStatus(_ *cli.Operator, args []string) error {
	fs := flag.NewFlagSet("dev status", flag.ContinueOnError)
	dir := fs.String("dir", "./devcluster", "cluster directory")
	if err := fs.Parse(args); err != nil {
		return err
	}
	c, err := Load(*dir)
	if err != nil {
		return err
	}
	w := tabwriter.NewWriter(os.Stdout, 2, 4, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tKIND\tPID\tALIVE\tLOG")
	for _, p := range c.Procs {
		fmt.Fprintf(w, "%s\t%s\t%d\t%v\t%s\n", p.Name, p.Kind, p.PID, syscall.Kill(p.PID, 0) == nil, p.Log)
	}
	return w.Flush()
}
