package controllers

import (
	"context"
	"strings"
	"time"

	srov1beta1 "github.com/openshift-psap/special-resource-operator/api/v1beta1"
	"github.com/openshift-psap/special-resource-operator/pkg/cache"
	"github.com/openshift-psap/special-resource-operator/pkg/clients"
	"github.com/openshift-psap/special-resource-operator/pkg/exit"
	"github.com/openshift-psap/special-resource-operator/pkg/osversion"
	"github.com/openshift-psap/special-resource-operator/pkg/warn"

	"github.com/pkg/errors"
	errs "github.com/pkg/errors"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	//machineV1 "github.com/openshift/machine-config-operator/pkg/apis/machineconfiguration.openshift.io/v1"
)

type ResourceGroupName struct {
	DriverBuild            string `json:"driverBuild"`
	DriverContainer        string `json:"driverContainer"`
	RuntimeEnablement      string `json:"runtimeEnablement"`
	DevicePlugin           string `json:"devicePlugin"`
	DeviceMonitoring       string `json:"deviceMonitoring"`
	DeviceDashboard        string `json:"deviceDashboard"`
	DeviceFeatureDiscovery string `json:"deviceFeatureDiscovery"`
	CSIDriver              string `json:"csiDriver"`
}

type ResourceStateName struct {
	DriverContainer   string
	RuntimeEnablement string
	DevicePlugin      string
	DeviceMonitoring  string
	/*
		"driver-container":   {"specialresource.openshift.io/driver-container-" + hw: "ready"},
		"runtime-enablement": {"specialresource.openshift.io/runtime-enablement-" + hw: "ready"},
		"device-plugin":      {"specialresource.openshift.io/device-plugin-" + hw: "ready"},
		"device-monitoring":  {"specialresource.openshift.io/device-monitoring-" + hw: "ready"},
	*/
}

type ProxyConfiguration struct {
	HttpProxy  string
	HttpsProxy string
	NoProxy    string
	TrustedCA  string
}

type RuntimeInformation struct {
	OperatingSystemMajor      string                        `json:"operatingSystemMajor"`
	OperatingSystemMajorMinor string                        `json:"operatingSystemMajorMinor"`
	OperatingSystemDecimal    string                        `json:"operatingSystemDecimal"`
	KernelFullVersion         string                        `json:"kernelFullVersion"`
	KernelPatchVersion        string                        `json:"kernelPatchVersion"`
	ClusterVersion            string                        `json:"clusterVersion"`
	ClusterVersionMajorMinor  string                        `json:"clusterVersionMajorMinor"`
	ClusterUpgradeInfo        map[string]NodeUpgradeVersion `json:"clusterUpgradeInfo"`
	UpdateVendor              string                        `json:"updateVendor"`
	PushSecretName            string                        `json:"pushSecretName"`
	OSImageURL                string                        `json:"osImageURL"`
	Proxy                     ProxyConfiguration            `json:"proxy"`
	GroupName                 ResourceGroupName             `json:"groupName"`
	StateName                 ResourceStateName             `json:"stateName"`
	SpecialResource           srov1beta1.SpecialResource    `json:"specialresource"`
}

var runInfo = RuntimeInformation{
	OperatingSystemMajor:      "",
	OperatingSystemMajorMinor: "",
	OperatingSystemDecimal:    "",
	KernelFullVersion:         "",
	KernelPatchVersion:        "",
	ClusterVersion:            "",
	ClusterVersionMajorMinor:  "",
	ClusterUpgradeInfo:        make(map[string]NodeUpgradeVersion),
	UpdateVendor:              "",
	PushSecretName:            "",
	OSImageURL:                "",
	Proxy:                     ProxyConfiguration{},
	GroupName: ResourceGroupName{
		DriverBuild:            "driver-build",
		DriverContainer:        "driver-container",
		RuntimeEnablement:      "runtime-enablement",
		DevicePlugin:           "device-plugin",
		DeviceMonitoring:       "device-monitoring",
		DeviceDashboard:        "device-dashboard",
		DeviceFeatureDiscovery: "device-feature-discovery",
		CSIDriver:              "csi-driver"},
	StateName: ResourceStateName{
		DriverContainer:   "specialresource.openshift.io/driver-container",
		RuntimeEnablement: "specialresource.openshift.io/runtime-enablement",
		DevicePlugin:      "specialresource.openshift.io/device-plugin",
		DeviceMonitoring:  "specialresource.openshift.io/device-monitoring"},
	SpecialResource: srov1beta1.SpecialResource{},
}

func logRuntimeInformation() {
	log.Info("Runtime Information", "OperatingSystemMajor", runInfo.OperatingSystemMajor)
	log.Info("Runtime Information", "OperatingSystemMajorMinor", runInfo.OperatingSystemMajorMinor)
	log.Info("Runtime Information", "OperatingSystemDecimal", runInfo.OperatingSystemDecimal)
	log.Info("Runtime Information", "KernelFullVersion", runInfo.KernelFullVersion)
	log.Info("Runtime Information", "KernelPatchVersion", runInfo.KernelPatchVersion)
	log.Info("Runtime Information", "ClusterVersion", runInfo.ClusterVersion)
	log.Info("Runtime Information", "ClusterVersionMajorMinor", runInfo.ClusterVersionMajorMinor)
	log.Info("Runtime Information", "ClusterUpgradeInfo", runInfo.ClusterUpgradeInfo)
	log.Info("Runtime Information", "UpdateVendor", runInfo.UpdateVendor)
	log.Info("Runtime Information", "PushSecretName", runInfo.PushSecretName)
	log.Info("Runtime Information", "OSImageURL", runInfo.OSImageURL)
	log.Info("Runtime Information", "Proxy", runInfo.Proxy)
}

func getRuntimeInformation(r *SpecialResourceReconciler) {

	var err error
	cache.Node.List, err = cacheNodes(r, false)
	exit.OnError(errs.Wrap(err, "Failed to cache nodes"))

	runInfo.OperatingSystemMajor, runInfo.OperatingSystemMajorMinor, runInfo.OperatingSystemDecimal, err = getOperatingSystem()
	exit.OnError(errs.Wrap(err, "Failed to get operating system"))

	runInfo.KernelFullVersion, err = getKernelFullVersion()
	exit.OnError(errs.Wrap(err, "Failed to get kernel version"))

	runInfo.KernelPatchVersion, err = getKernelPatchVersion()
	exit.OnError(errs.Wrap(err, "Failed to get kernel patch version"))

	runInfo.ClusterVersion, runInfo.ClusterVersionMajorMinor, err = getClusterVersion(r)
	exit.OnError(errs.Wrap(err, "Failed to get cluster version"))

	runInfo.ClusterUpgradeInfo, err = getUpgradeInfo()
	exit.OnError(errs.Wrap(err, "Failed to get upgrade info"))

	runInfo.PushSecretName, err = retryGetPushSecretName(r)
	exit.OnError(errs.Wrap(err, "Failed to get push secret name"))

	runInfo.OSImageURL, err = getOSImageURL(r)
	exit.OnError(errs.Wrap(err, "Failed to get OSImageURL"))

	runInfo.Proxy, err = getProxyConfiguration(r)
	exit.OnError(errs.Wrap(err, "Failed to get Proxy Configuration"))

	r.specialresource.DeepCopyInto(&runInfo.SpecialResource)
}

func getOperatingSystem() (string, string, string, error) {

	var nodeOSrel string
	var nodeOSmaj string
	var nodeOSmin string

	// Assuming all nodes are running the same os
	os := "feature.node.kubernetes.io/system-os_release"

	for _, node := range cache.Node.List.Items {
		labels := node.GetLabels()
		nodeOSrel = labels[os+".ID"]
		nodeOSmaj = labels[os+".VERSION_ID.major"]
		nodeOSmin = labels[os+".VERSION_ID.minor"]

		if len(nodeOSrel) == 0 || len(nodeOSmaj) == 0 {
			return "", "", "", errs.New("Cannot extract " + os + ".*, is NFD running? Check node labels")
		}
	}

	return osversion.RenderOperatingSystem(nodeOSrel, nodeOSmaj, nodeOSmin)
}

func getKernelFullVersion() (string, error) {

	var found bool
	var kernelFullVersion string
	// Assuming all nodes are running the same kernel version,
	// one could easily add driver-kernel-versions for each node.
	for _, node := range cache.Node.List.Items {
		labels := node.GetLabels()

		// We only need to check for the key, the value
		// is available if the key is there
		short := "feature.node.kubernetes.io/kernel-version.full"
		if kernelFullVersion, found = labels[short]; !found {
			return "", errs.New("Label " + short + " not found is NFD running? Check node labels")
		}
	}

	return kernelFullVersion, nil
}

// Using w.xx.y-zzz and looking at the fourth file listed /boot/vmlinuz-4.4.0-45 we can say:
// w = Kernel Version = 4
// xx= Major Revision = 4
// y = Minor Revision = 0
// zzz=Patch number = 45
func getKernelPatchVersion() (string, error) {

	version := strings.Split(runInfo.KernelFullVersion, "-")
	// Happens only if kernel full version has no patch version sep by "-"
	if len(version) == 1 {
		short := strings.Split(runInfo.KernelFullVersion, ".")
		return short[0] + "." + short[1] + "." + short[2], nil
	}

	patch := strings.Split(version[1], ".")
	// version.major.minor-patch
	return version[0] + "-" + patch[0], nil
}

func getClusterVersion(r *SpecialResourceReconciler) (string, string, error) {

	version, err := clients.Interface.ClusterVersions().Get(context.TODO(), "version", metav1.GetOptions{})
	if err != nil {
		return "", "", errs.Wrap(err, "ConfigClient unable to get ClusterVersions")
	}

	var majorMinor string
	for _, condition := range version.Status.History {
		if condition.State != "Completed" {
			continue
		}

		s := strings.Split(condition.Version, ".")

		if len(s) > 1 {
			majorMinor = s[0] + "." + s[1]
		} else {
			majorMinor = s[0]
		}

		return condition.Version, majorMinor, nil
	}

	return "", "", errs.New("Undefined Cluster Version")
}

func retryGetPushSecretName(r *SpecialResourceReconciler) (string, error) {
	for i := 0; i < 3; i++ {
		time.Sleep(2 * time.Second)
		pushSecretName, err := getPushSecretName(r)
		if err != nil {
			log.Info("Cannot find Secret builder-dockercfg " + r.specialresource.Spec.Namespace)
			continue
		} else {
			return pushSecretName, err
		}
	}

	return "", errors.New("Cannot find Secret builder-dockercfg")

}

func getPushSecretName(r *SpecialResourceReconciler) (string, error) {

	secrets := &unstructured.UnstructuredList{}

	secrets.SetAPIVersion("v1")
	secrets.SetKind("SecretList")

	log.Info("Getting SecretList in Namespace: " + r.specialresource.Spec.Namespace)
	opts := []client.ListOption{
		client.InNamespace(r.specialresource.Spec.Namespace),
	}
	err := clients.Interface.List(context.TODO(), secrets, opts...)
	if err != nil {
		return "", errors.Wrap(err, "Client cannot get SecretList")
	}

	log.Info("Searching for builder-dockercfg Secret")
	for _, secret := range secrets.Items {
		secretName := secret.GetName()

		if strings.Contains(secretName, "builder-dockercfg") {
			log.Info("Found", "Secret", secretName)
			return secretName, nil
		}
	}

	return "", errors.New("Cannot find Secret builder-dockercfg")
}

func getOSImageURL(r *SpecialResourceReconciler) (string, error) {

	cm := &unstructured.Unstructured{}
	cm.SetAPIVersion("v1")
	cm.SetKind("ConfigMap")

	namespacedName := types.NamespacedName{Namespace: "openshift-machine-config-operator", Name: "machine-config-osimageurl"}
	err := clients.Interface.Get(context.TODO(), namespacedName, cm)
	if apierrors.IsNotFound(err) {
		return "", errs.Wrap(err, "ConfigMap machine-config-osimageurl -n  openshift-machine-config-operator not found")
	}

	osImageURL, found, err := unstructured.NestedString(cm.Object, "data", "osImageURL")
	exit.OnErrorOrNotFound(found, err)

	return osImageURL, nil

}

func getProxyConfiguration(r *SpecialResourceReconciler) (ProxyConfiguration, error) {

	proxy := ProxyConfiguration{}

	cfgs := &unstructured.UnstructuredList{}
	cfgs.SetAPIVersion("config.openshift.io/v1")
	cfgs.SetKind("ProxyList")

	opts := []client.ListOption{}

	err := clients.Interface.List(context.TODO(), cfgs, opts...)
	if err != nil {
		return proxy, errors.Wrap(err, "Client cannot get ProxyList")
	}

	for _, cfg := range cfgs.Items {
		cfgName := cfg.GetName()

		var fnd bool
		var err error
		// If no proxy is configured, we do not exit we just give a warning
		// and initialized the Proxy struct with zero sized strings
		if strings.Contains(cfgName, "cluster") {
			if proxy.HttpProxy, fnd, err = unstructured.NestedString(cfg.Object, "spec", "httpProxy"); err != nil {
				warn.OnErrorOrNotFound(fnd, err)
				proxy.HttpProxy = ""
			}

			if proxy.HttpsProxy, fnd, err = unstructured.NestedString(cfg.Object, "spec", "httpsProxy"); err != nil {
				warn.OnErrorOrNotFound(fnd, err)
				proxy.HttpsProxy = ""
			}

			if proxy.NoProxy, fnd, err = unstructured.NestedString(cfg.Object, "spec", "noProxy"); err != nil {
				warn.OnErrorOrNotFound(fnd, err)
				proxy.NoProxy = ""
			}

			if proxy.TrustedCA, fnd, err = unstructured.NestedString(cfg.Object, "spec", "trustedCA", "name"); err != nil {
				warn.OnErrorOrNotFound(fnd, err)
				proxy.TrustedCA = ""
			}
		}
	}

	return proxy, nil
}

func setupProxy(obj *unstructured.Unstructured, r *SpecialResourceReconciler) error {

	if strings.Compare(obj.GetKind(), "Pod") == 0 {
		if err := setupPodProxy(obj, r); err != nil {
			return errs.Wrap(err, "Cannot setup Pod Proxy")
		}
	}
	if strings.Compare(obj.GetKind(), "DaemonSet") == 0 {
		if err := setupDaemonSetProxy(obj, r); err != nil {
			return errs.Wrap(err, "Cannot setup DaemonSet Proxy")
		}

	}

	return nil
}

// We may generalize more depending on how many entities need proxy settings.
// path... -> Pod, DaemonSet, BuildConfig, etc.
func setupDaemonSetProxy(obj *unstructured.Unstructured, r *SpecialResourceReconciler) error {
	containers, found, err := unstructured.NestedSlice(obj.Object, "spec", "template", "spec", "containers")
	exit.OnErrorOrNotFound(found, err)

	if err := setupContainersProxy(containers); err != nil {
		return errs.Wrap(err, "Cannot set proxy for Pod")
	}

	return nil
}

func setupPodProxy(obj *unstructured.Unstructured, r *SpecialResourceReconciler) error {

	containers, found, err := unstructured.NestedSlice(obj.Object, "spec", "containers")
	exit.OnErrorOrNotFound(found, err)

	if err := setupContainersProxy(containers); err != nil {
		return errs.Wrap(err, "Cannot set proxy for Pod")
	}

	return nil
}

func setupContainersProxy(containers []interface{}) error {

	for _, container := range containers {
		switch container := container.(type) {
		case map[string]interface{}:
			env, found, err := unstructured.NestedSlice(container, "env")
			exit.OnError(err)

			// If env not found we are creating a new env slice
			// otherwise we're appending it to the existing env slice
			httpproxy := make(map[string]interface{})
			httpsproxy := make(map[string]interface{})
			noproxy := make(map[string]interface{})

			httpproxy["name"] = "HTTP_PROXY"
			httpproxy["value"] = runInfo.Proxy.HttpProxy

			httpsproxy["name"] = "HTTPS_PROXY"
			httpsproxy["value"] = runInfo.Proxy.HttpsProxy

			noproxy["name"] = "NO_PROXY"
			noproxy["value"] = runInfo.Proxy.NoProxy

			if !found {
				env = make([]interface{}, 0)
			}

			env = append(env, httpproxy)
			env = append(env, httpsproxy)
			env = append(env, noproxy)

			if err := unstructured.SetNestedSlice(container, env, "env"); err != nil {
				return errs.Wrap(err, "Cannot set env for container")
			}

		default:
			log.Info("container", "DEFAULT NOT THE CORRECT TYPE", container)
		}
		break
	}

	return nil
}
