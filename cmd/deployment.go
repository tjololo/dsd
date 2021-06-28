/*
Copyright © 2021 NAME HERE <EMAIL ADDRESS>

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
package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/tjololo/dsd/pkg/sidecar"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

var (
	containername string
	namespace string
	kubeconfig string
	debugimage string
)

// deploymentCmd represents the deployment command
var deploymentCmd = &cobra.Command{
	Use:   "deployment [deployment]",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Args: cobra.ExactArgs(1),
	Run: UpdateDeployment,
}

func init() {
	debugCmd.AddCommand(deploymentCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// deploymentCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// deploymentCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
	deploymentCmd.Flags().StringVarP(&containername, "container", "c" , "", "Supply container name if deployment contains multiple pods")
	deploymentCmd.Flags().StringVarP(&namespace, "namespace", "n", "default", "Namepsace of the deployment to debug")
	deploymentCmd.Flags().StringVar(&kubeconfig, "kubeconfig", "", "absolute path to the kubeconfig")
	deploymentCmd.Flags().StringVar(&debugimage, "debugimage", "tjololo/dotnet-debug:alpine", "image to add as a debug sidecar")
}


func UpdateDeployment(cmd *cobra.Command, args []string) {
	if home := homedir.HomeDir(); home != "" && kubeconfig == "" {
		kubeconfig = filepath.Join(home, ".kube", "config")
	}
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		panic(err.Error())
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}
	h := sidecar.Helper{
		Client: clientset,
	}
	d, err := h.AddDebugSidecar(cmd.Context(), namespace, args[0], containername, debugimage)
	if err != nil {
		panic(err.Error())
	}
	fmt.Printf("Added sidecar to deployment % with uid %s\n", d.Name, d.UID)
}