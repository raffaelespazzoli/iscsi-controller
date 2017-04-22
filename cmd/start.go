// Copyright © 2017 NAME HERE <EMAIL ADDRESS>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"github.com/Sirupsen/logrus"
	"github.com/kubernetes-incubator/external-storage/lib/controller"
	"github.com/raffaelespazzoli/iscsi-controller/provisioner"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"time"
)

var log = logrus.New()

// start-controllerCmd represents the start-controller command
var startcontrollerCmd = &cobra.Command{
	Use:   "start",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		initLog()

		log.Debugln("start called")

		// creates the in-cluster config
		log.Debugln("creating in cluster default kube client config")
		config, err := rest.InClusterConfig()
		if err != nil {
			log.Fatalln(err)
		}
		log.WithFields(logrus.Fields{
			"config-host": config.Host,
		}).Debugln("kube client config created")

		//	 creates the clientset
		log.Debugln("creating kube client set")
		kubernetesClientSet, err := kubernetes.NewForConfig(config)
		if err != nil {
			log.Fatalln(err)
		}
		log.Debugln("kube client set created")

		// The controller needs to know what the server version is because out-of-tree
		// provisioners aren't officially supported until 1.5
		serverVersion, err := kubernetesClientSet.Discovery().ServerVersion()
		if err != nil {
			log.Fatalf("Error getting server version: %v", err)
		}

		iscsiProvisioner := provisioner.NewiscsiProvisioner(viper.GetString("targetd-url"), viper.GetString("pool-name"), viper.GetString("initiator-wwn"))
		pc := controller.NewProvisionController(kubernetesClientSet, viper.GetDuration("resync-period"), viper.GetString("provisioner-name"), iscsiProvisioner, serverVersion.GitVersion,
			viper.GetBool("exponential-backoff-on-error"), viper.GetInt("fail-retry-threshold"), viper.GetDuration("lease-period"),
			viper.GetDuration("renew-deadline"), viper.GetDuration("retry-priod"), viper.GetDuration("term-limit"))
		pc.Run(wait.NeverStop)
	},
}

func init() {
	RootCmd.AddCommand(startcontrollerCmd)
	startcontrollerCmd.Flags().String("provisioner-name", "iscsi-provisioner", "name of this provisioner, must match what is passed int the storage class annotation")
	viper.BindPFlag("provisioner-name", startcontrollerCmd.Flags().Lookup("provisioner-name"))
	startcontrollerCmd.Flags().Duration("resync-period", 15*time.Second, "how often to poll the master API for updates")
	viper.BindPFlag("resync-period", startcontrollerCmd.Flags().Lookup("resync-period"))
	startcontrollerCmd.Flags().Bool("exponential-backoff-on-error", true, "")
	viper.BindPFlag("exponential-backoff-on-error", startcontrollerCmd.Flags().Lookup("exponential-backoff-on-error"))
	startcontrollerCmd.Flags().Int("fail-retry-threshold", 10, "Threshold for max number of retries on failure of provisioner")
	viper.BindPFlag("fail-retry-threshold", startcontrollerCmd.Flags().Lookup("fail-retry-threshold"))
	startcontrollerCmd.Flags().Duration("lease-period", controller.DefaultLeaseDuration, "LeaseDuration is the duration that non-leader candidates will wait to force acquire leadership. This is measured against time of last observed ack")
	viper.BindPFlag("lease-period", startcontrollerCmd.Flags().Lookup("lease-period"))
	startcontrollerCmd.Flags().Duration("renew-deadline", controller.DefaultRenewDeadline, "RenewDeadline is the duration that the acting master will retry refreshing leadership before giving up")
	viper.BindPFlag("renew-deadline", startcontrollerCmd.Flags().Lookup("renew-deadline"))
	startcontrollerCmd.Flags().Duration("retry-priod", controller.DefaultRetryPeriod, "RetryPeriod is the duration the LeaderElector clients should wait between tries of actions")
	viper.BindPFlag("retry-priod", startcontrollerCmd.Flags().Lookup("retry-priod"))
	startcontrollerCmd.Flags().Duration("term-limit", controller.DefaultTermLimit, "TermLimit is the maximum duration that a leader may remain the leader to complete the task before it must give up its leadership. 0 for forever or indefinite.")
	viper.BindPFlag("term-limit", startcontrollerCmd.Flags().Lookup("term-limit"))
	startcontrollerCmd.Flags().String("targetd-url", "localhost:18700", "iscsi targetd endpoint url.")
	viper.BindPFlag("targetd-url", startcontrollerCmd.Flags().Lookup("targetd-url"))
	startcontrollerCmd.Flags().String("pool-name", "openshift-pool", "name of the logical volume pool to be useto create volume (make sure it's large enough).")
	viper.BindPFlag("pool-name", startcontrollerCmd.Flags().Lookup("pool-name"))
	startcontrollerCmd.Flags().String("initiator-wwn", "openshift-initiator", "World wide name of the initiator")
	viper.BindPFlag("initiator-wwn", startcontrollerCmd.Flags().Lookup("initiator-wwn"))

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// start-controllerCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// start-controllerCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")

}

func initLog() {
	var err error
	log.Level, err = logrus.ParseLevel(viper.GetString("log-level"))
	if err != nil {
		log.Fatalln(err)
	}
}
