package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/schollz/progressbar/v3"
)

var active_resources []string
var namespace string
var destination string

func error_handler(msg string, err error) {
	fmt.Fprintf(os.Stderr, "%s: %v", msg, err)
	os.Exit(1)
}

func checkResource(resource string) bool {
	cmd := exec.Command("kubectl", "get", resource, "--no-headers")
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = nil

	if err := cmd.Run(); err != nil {
		return false
	}

	return len(stdout.String()) > 0
}

func getAPIResources() []string {
	cmd := exec.Command("kubectl", "api-resources", "--verbs=list", "--namespaced=true", "-n", namespace, "-o", "name")
	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		error_handler("Error while getting API resources", err)
	}

	return strings.Fields(stdout.String())
}

func create_directory(path string) {
	// Ensure proper path formatting for Windows
	cleanPath := filepath.FromSlash(path)

	// Check if directory already exists
	if _, err := os.Stat(cleanPath); os.IsNotExist(err) {
		err := os.MkdirAll(cleanPath, 0755)
		if err != nil {
			error_handler(fmt.Sprintf("Failed to create directory %s: ", cleanPath), err)
		}
	} else {
		error_handler(fmt.Sprintf("Directory already exists %s: ", cleanPath), err)
	}
}

func get_resource(path, resource string) {
	filename := "manifest_" + resource + ".yaml"
	cmd := exec.Command("kubectl", "get", resource, "-o", "yaml")

	cmd.Dir = path
	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		error_handler(fmt.Sprintf("Error while getting the resource %s: ", resource), err)
	}

	// Write the output to the file
	cleanpath := filepath.Join(path, filename)
	if err := os.WriteFile(cleanpath, stdout.Bytes(), 0644); err != nil {
		error_handler(fmt.Sprintf("Error writing to file %s: ", filename), err)
	}

}

func removeDuplicatesAndEmpty(slice []string, ns_path string) {
	// using map to record duplicates
	encountered := map[string]bool{}
	count := 0

	// create a file
	filePath := filepath.Join(ns_path, "docker_images.txt")
	active_file, err := os.Create(filePath)
	if err != nil {
		error_handler("Error while creating a file", err)
	}

	for _, str := range slice {
		if str == "" {
			continue
		}
		if !encountered[str] {
			count++
			encountered[str] = true

			_, err := active_file.WriteString(str + "\n")
			if err != nil {
				error_handler("Error while writing to a file", err)
			}
		}
	}

	fmt.Println("Total Docker Images in use: ", count)

}

func grab_images(ns_path string) {
	var allImages []string

	// you can add or remove the places where you want to check
	res_types := []string{
		"deployments",
		"statefulsets",
		"daemonsets",
		"cronjobs",
		"jobs",
		"pods",
		"replicasets",
	}

	// Build jsonpath query for both containers and initContainers
	jsonPath := "{.items[*].spec.template.spec.containers[*].image} {.items[*].spec.template.spec.initContainers[*].image}"

	for _, resource := range res_types {

		// Construct kubectl command
		cmd := exec.Command("kubectl", "get", resource, "-n", namespace, "-o", fmt.Sprintf("jsonpath=%s", jsonPath))

		// Capture command output
		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		err := cmd.Run()
		if err != nil {
			// Skip if resource type doesn't exist in the namespace
			if strings.Contains(stderr.String(), "not found") {
				continue
			}
			// not using error_handler because we still want to continue
			fmt.Printf("\nError getting images for %s: %v - %s \nContinuing the execution ...\n", resource, err, stderr.String())
		}

		if stdout.Len() > 0 {
			images := strings.Fields(stdout.String())
			allImages = append(allImages, images...)
		}
	}

	removeDuplicatesAndEmpty(allImages, ns_path)
}

func main() {

	dest := flag.String("destination", "", "Destination directory/folder to save.")
	ns := flag.String("namespace", "", "Provide namespace.")

	flag.Parse()

	destination = *dest
	namespace = *ns

	if destination == "" {
		flag.Usage()
		os.Exit(0)
	}

	_, err := os.Stat(destination)
	if err != nil {
		error_handler("Give directory does not exist!", err)
	}

	if namespace == "" {
		cmd := exec.Command("kubectl", "config", "view", "--minify", "-o", "jsonpath={..namespace}")

		var stdout bytes.Buffer
		cmd.Stdout = &stdout

		if err := cmd.Run(); err != nil {
			error_handler("Error getting the current context's namespace", err)
		}

		namespace = stdout.String()
		if stdout.String() == "" {
			error_handler("No default namespace found in your current context", nil)
		} else {
			fmt.Println("Taking current context's namespace:", namespace)
		}

	}

	// create initial namespace directory
	ns_path := filepath.Join(destination, namespace)
	create_directory(ns_path)

	grab_images(ns_path)

	// create active_resources.txt file
	filePath := filepath.Join(ns_path, "active_resources.txt")
	active_res_file, err := os.Create(filePath)
	if err != nil {
		error_handler("Error while creating a file", err)
	}

	resources := getAPIResources()
	total := len(resources)
	fmt.Printf("Total resources %s namespace supports: %v\n\n", namespace, total)

	activeCount := 0

	bar := progressbar.Default(int64(total))

	for _, resource := range resources {
		bar.Add(1)
		if checkResource(resource) {
			activeCount++

			active_resources = append(active_resources, resource)

			_, err := active_res_file.WriteString(resource + "\n")
			if err != nil {
				error_handler("Error while writing to a file", err)
			}

			// create resource directory
			res_path := filepath.Join(ns_path, resource)
			create_directory(res_path)

			get_resource(res_path, resource)
		}
	}

	fmt.Printf("\nTotal Active Resources: %d/%d", activeCount, total)
	fmt.Printf("\nAll the results are stored at %s\n", ns_path)
	defer active_res_file.Close()

}
