package gitreceive

import (
	"fmt"
	"strings"
	"testing"

	"k8s.io/kubernetes/pkg/api"
)

func TestDockerBuilderPodName(t *testing.T) {
	name := dockerBuilderPodName("demo", "12345678")
	if !strings.HasPrefix(name, "dockerbuild-demo-12345678-") {
		t.Fatalf("expected pod name dockerbuild-demo-12345678-*, got %s", name)
	}
}

func TestSlugBuilderPodName(t *testing.T) {
	name := slugBuilderPodName("demo", "12345678")
	if !strings.HasPrefix(name, "slugbuild-demo-12345678-") {
		t.Fatalf("expected pod name slugbuild-demo-12345678-*, got %s", name)
	}
}

type slugBuildCase struct {
	debug                      bool
	name                       string
	namespace                  string
	env                        map[string]interface{}
	tarKey                     string
	putKey                     string
	cacheKey                   string
	gitShortHash               string
	buildPack                  string
	slugBuilderImage           string
	slugBuilderImagePullPolicy api.PullPolicy
	storageType                string
}

type dockerBuildCase struct {
	debug                        bool
	name                         string
	namespace                    string
	env                          map[string]interface{}
	tarKey                       string
	gitShortHash                 string
	imgName                      string
	dockerBuilderImage           string
	dockerBuilderImagePullPolicy api.PullPolicy
	storageType                  string
}

func TestBuildPod(t *testing.T) {
	emptyEnv := make(map[string]interface{})

	env := make(map[string]interface{})
	env["KEY"] = "VALUE"

	var pod *api.Pod

	slugBuilds := []slugBuildCase{
		{true, "test", "default", emptyEnv, "tar", "put-url", "cache-url", "deadbeef", "", "", api.PullAlways, ""},
		{true, "test", "default", env, "tar", "put-url", "cache-url", "deadbeef", "", "", api.PullAlways, ""},
		{true, "test", "default", emptyEnv, "tar", "put-url", "", "deadbeef", "", "", api.PullAlways, ""},
		{true, "test", "default", emptyEnv, "tar", "put-url", "cache-url", "deadbeef", "buildpack", "", api.PullAlways, ""},
		{true, "test", "default", env, "tar", "put-url", "cache-url", "deadbeef", "buildpack", "", api.PullAlways, ""},
		{true, "test", "default", env, "tar", "put-url", "cache-url", "deadbeef", "buildpack", "customimage", api.PullAlways, ""},
		{true, "test", "default", env, "tar", "put-url", "cache-url", "deadbeef", "buildpack", "customimage", api.PullIfNotPresent, ""},
		{true, "test", "default", env, "tar", "put-url", "cache-url", "deadbeef", "buildpack", "customimage", api.PullNever, ""},
	}

	for _, build := range slugBuilds {
		pod = slugbuilderPod(
			build.debug,
			build.name,
			build.namespace,
			build.env,
			build.tarKey,
			build.putKey,
			build.cacheKey,
			build.gitShortHash,
			build.buildPack,
			build.storageType,
			build.slugBuilderImage,
			build.slugBuilderImagePullPolicy,
		)

		if pod.ObjectMeta.Name != build.name {
			t.Errorf("expected %v but returned %v ", build.name, pod.ObjectMeta.Name)
		}

		if pod.ObjectMeta.Namespace != build.namespace {
			t.Errorf("expected %v but returned %v ", build.namespace, pod.ObjectMeta.Namespace)
		}

		checkForEnv(t, pod, "SOURCE_VERSION", build.gitShortHash)
		checkForEnv(t, pod, "TAR_PATH", build.tarKey)
		checkForEnv(t, pod, "PUT_PATH", build.putKey)

		if build.cacheKey == "" {
			if cachePath, err := envValueFromKey(pod, "CACHE_PATH"); err == nil {
				t.Errorf("expected CACHE_PATH not to be defined but it was defined with %v", cachePath)
			}
		} else {
			checkForEnv(t, pod, "CACHE_PATH", build.cacheKey)
		}

		if build.buildPack != "" {
			checkForEnv(t, pod, "BUILDPACK_URL", build.buildPack)
		}

		if build.slugBuilderImage != "" {
			if pod.Spec.Containers[0].Image != build.slugBuilderImage {
				t.Errorf("expected %v but returned %v ", build.slugBuilderImage, pod.Spec.Containers[0].Image)
			}
		}
		if build.slugBuilderImagePullPolicy != "" {
			if pod.Spec.Containers[0].ImagePullPolicy != build.slugBuilderImagePullPolicy {
				t.Errorf("expected %v but returned %v", build.slugBuilderImagePullPolicy, pod.Spec.Containers[0].ImagePullPolicy)
			}
		}
	}

	dockerBuilds := []dockerBuildCase{
		{true, "test", "default", emptyEnv, "tar", "deadbeef", "", "", api.PullAlways, ""},
		{true, "test", "default", env, "tar", "deadbeef", "", "", api.PullAlways, ""},
		{true, "test", "default", emptyEnv, "tar", "deadbeef", "img", "", api.PullAlways, ""},
		{true, "test", "default", env, "tar", "deadbeef", "img", "", api.PullAlways, ""},
		{true, "test", "default", env, "tar", "deadbeef", "img", "customimage", api.PullAlways, ""},
		{true, "test", "default", env, "tar", "deadbeef", "img", "customimage", api.PullIfNotPresent, ""},
		{true, "test", "default", env, "tar", "deadbeef", "img", "customimage", api.PullNever, ""},
	}
	regEnv := map[string]string{"REG_LOC": "on-cluster"}
	for _, build := range dockerBuilds {
		pod = dockerBuilderPod(
			build.debug,
			build.name,
			build.namespace,
			build.env,
			build.tarKey,
			build.gitShortHash,
			build.imgName,
			build.storageType,
			build.dockerBuilderImage,
			"localhost",
			"5555",
			regEnv,
			build.dockerBuilderImagePullPolicy,
		)

		if pod.ObjectMeta.Name != build.name {
			t.Errorf("expected %v but returned %v ", build.name, pod.ObjectMeta.Name)
		}
		if pod.ObjectMeta.Namespace != build.namespace {
			t.Errorf("expected %v but returned %v ", build.namespace, pod.ObjectMeta.Namespace)
		}

		checkForEnv(t, pod, "SOURCE_VERSION", build.gitShortHash)
		checkForEnv(t, pod, "TAR_PATH", build.tarKey)
		checkForEnv(t, pod, "IMG_NAME", build.imgName)
		checkForEnv(t, pod, "REG_LOC", "on-cluster")
		if build.dockerBuilderImage != "" {
			if pod.Spec.Containers[0].Image != build.dockerBuilderImage {
				t.Errorf("expected %v but returned %v", build.dockerBuilderImage, pod.Spec.Containers[0].Image)
			}
		}
		if build.dockerBuilderImagePullPolicy != "" {
			if pod.Spec.Containers[0].ImagePullPolicy != "" {
				if pod.Spec.Containers[0].ImagePullPolicy != build.dockerBuilderImagePullPolicy {
					t.Errorf("expected %v but returned %v", build.dockerBuilderImagePullPolicy, pod.Spec.Containers[0].ImagePullPolicy)
				}
			}
		}
	}
}

func checkForEnv(t *testing.T, pod *api.Pod, key, expVal string) {
	val, err := envValueFromKey(pod, key)
	if err != nil {
		t.Errorf("%v", err)
	}
	if val != val {
		t.Errorf("expected %v but returned %v ", expVal, val)
	}
}

func envValueFromKey(pod *api.Pod, key string) (string, error) {
	for _, env := range pod.Spec.Containers[0].Env {
		if env.Name == key {
			return env.Value, nil
		}
	}

	return "", fmt.Errorf("no key with name %v found in pod env", key)
}
