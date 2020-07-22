package cmd

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/appvia/artefactor/pkg/docker"
	"github.com/spf13/cobra"
)

const (
	// ImageNamesCommand is the sub command syntax
	ImageNamesCommand string = "update-image-vars"
)

// imageNamesCmd represents the command to display image information
var imageNamesCmd = &cobra.Command{
	Use:   ImageNamesCommand,
	Short: "update docker image name variables",
	Long:  "update docker image names from environment with registry name",
	RunE: func(c *cobra.Command, args []string) error {
		return imageNames(c)
	},
}

func init() {
	addFlagWithEnvDefault(
		imageNamesCmd,
		FlagArchiveDir,
		DefaultArchiveDir,
		"a directory where artefacts exist to publish from")

	addFlagWithEnvDefault(
		imageNamesCmd,
		FlagImageVars,
		"",
		"the whitelist separated list of variables specifying original image names")

	addFlagWithEnvDefault(
		imageNamesCmd,
		FlagDockerRegistry,
		"",
		"where images have been published e.g. private-registry.local")

	addFlagWithEnvDefault(
		imageNamesCmd,
		FlagDockerUserName,
		"",
		FlagDockerUserNameHelp)

	addFlagWithEnvDefault(
		imageNamesCmd,
		FlagDockerPassword,
		"",
		FlagDockerPasswordHelp)


	RootCmd.AddCommand(imageNamesCmd)
}

func imageNames(c *cobra.Command) error {
	common(c)
	// get the registry (if specified)
	registry := c.Flag(FlagDockerRegistry).Value.String()
	// Complain if no registry is specified
	if len(registry) < 1 {
		return fmt.Errorf("must specify registry for %s", ImageNamesCommand)
	}
	imageVars := strings.Fields(c.Flag(FlagImageVars).Value.String())

	for _, imageVar := range imageVars {
		image := os.Getenv(imageVar)
		//if the image has a sha, we need to check for a local sha
		log.Printf("Updating imagevar %s: %s", imageVar, image)
		newImageName := docker.GetNewImageName(image, registry)
		imageOrigSha := docker.GetRepoDigest(newImageName)
		if imageOrigSha != "" {
			newBareImageName := docker.StripRepoDigest(newImageName)
			if docker.GetImageTag(newBareImageName) == "" {
				newBareImageName = newBareImageName + ":repoDigest-" + imageOrigSha
			}

			//we have a sha, find new local sha
			repoDigests, err := docker.GetClientRepoDigestsByRegistry(newBareImageName, registry)
			if err != nil {
				if docker.IsClientErrNotFound(err) {
                    digest, err := docker.GetImageDigestFromRemoteRegistry(newBareImageName, getCredsFromFlags(c))
                    if err != nil {
				        return fmt.Errorf("No repoDigest matching %s found locally or in target registry. Please re run artefactor publish to upload and generate a repoDigest for this image in the target environment", newBareImageName)
                    }
                    repoDigests = append(repoDigests, digest)
				} else { return err }
			}
			if len(repoDigests) < 1 {
		        return fmt.Errorf("No repoDigest matching %s found locally or in target registry. Please re run artefactor publish to upload and generate a repoDigest for this image in the target environment", newBareImageName)
			} else if len(repoDigests) > 1 {
				// multiple version of image may cause this? defensive code: Not sure if this will ever happen
				return fmt.Errorf("Ambiguous repoDigests for image: %s, found multiple repo Digests attached to docker image:  %#v, Require unambiguous number of repoDigests for the image", image, repoDigests)
			}
			newImageName = docker.StripRepoDigest(newImageName) + docker.ShaIdent + docker.GetRepoDigest(repoDigests[0])
		}

		fmt.Printf("export %s=%s\n", imageVar, newImageName)
	}
	return nil
}
