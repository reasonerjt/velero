/*
Copyright 2017 the Velero contributors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package debug

import (
	_ "embed"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	crashdCmd "github.com/vmware-tanzu/crash-diagnostics/cmd"
	"github.com/vmware-tanzu/velero/pkg/client"
	"github.com/vmware-tanzu/velero/pkg/cmd"
)

//go:embed velero.cshd
var scriptBytes []byte

type option struct {
	// workdir for crashd will be $baseDir/velero-debug
	baseDir string
	// the namespace where velero server is installed
	namespace string
	// the absolute path for the log bundle to be generated
	outputPath string
	// the absolute path for the kubeconfig file that will be read by crashd for calling K8S API
	kubeconfigPath string
	// optional, the name of the backup resource whose log will be packaged into the debug bundle
	backup string
	// optional, the name of the restore resource whose log will be packaged into the debug bundle
	restore string
}

func (o *option) bindFlags(flags *pflag.FlagSet) {
	flags.StringVar(&o.outputPath, "output","", "The path of the bundle tarball, by default it's $HOME/bundle.tar.gz. Optional")
	flags.StringVar(&o.backup, "backup","","The name of the backup resource whose log will be collected, no backup logs will be collected if it's not set. Optional")
	flags.StringVar(&o.restore, "restore","","The name of the restore resource whose log will be collected, no restore logs will be collected if it's not set. Optional")
}

func (o *option) asCrashdArgs() string {
	return fmt.Sprintf("output=%s,namespace=%s,basedir=%s,backup=%s,restore=%s,kubeconfig=%s",
		o.outputPath, o.namespace, o.baseDir, o.backup, o.restore, o.kubeconfigPath)
}

func (o *option) complete(f client.Factory) error {
	if len(o.outputPath) > 0 {
		absOutputPath, err := filepath.Abs(o.outputPath)
		if err != nil {
			return fmt.Errorf("invalid output path: %v", err)
		}
		o.outputPath=absOutputPath
	}
	tmpDir, err := ioutil.TempDir("", "crashd")
	if err != nil {
		return err
	}
	o.baseDir=tmpDir
	o.namespace = f.Namespace()
	o.kubeconfigPath = kubeconfig(tmpDir)
	return nil
}

// NewCommand creates a cobra command.
func NewCommand(f client.Factory) *cobra.Command {
	o := &option{}
	c := &cobra.Command{
		Use:   "debug",
		Short: "Generate debug bundle",
		Long:  `TBD`,
		Run: func(c *cobra.Command, args []string) {
			defer func(opt *option)	{
				if len(o.baseDir) > 0 {
					if err := os.RemoveAll(o.baseDir); err != nil {
						fmt.Fprintf(os.Stderr, "Failed to remove temp dir: %s: %v", o.baseDir, err)
					}
				}
			}(o)
			err := o.complete(f)
			cmd.CheckError(err)
			err2 := runCrashd(o.asCrashdArgs())
			cmd.CheckError(err2)
			fmt.Println(os.Args)
		},
	}
	o.bindFlags(c.Flags())

	return c
}

func runCrashd(argString string) error {
	bak := os.Args
	defer func() {os.Args = bak}()
	f, err := ioutil.TempFile("", "velero*.cshd")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	_, err2 := f.Write(scriptBytes)
	if err2 != nil {
		return err2
	}
	os.Args = []string{"", "run", "--debug", f.Name(), "--args", fmt.Sprintf("%s", argString)}
	return crashdCmd.Run()
}

// TODO generate the temp file
func kubeconfig(dir string) string {
	return "/Users/jiangd/.kube/minikube-250-224/config"
}