package helpers

import (
	"fmt"
	"os"

	"github.com/go-git/go-git/v5"
)

// CheckDst checks if specified destination folder exists
func CheckDst(dst string) error {
	/* TO-DO: This function verifies if the directory exists and
	if it already exists, then it deletes the folder and its contents
	*/
	_, err := os.Stat(dst)
	if err == nil {
		fmt.Println("Folder and contents do exist, therefore deleting the folder and its contents ...")
		// delete folder and contents
		err := os.RemoveAll(dst)
		if err != nil {
			return err
		}
	} else {
		fmt.Println("Folder and contents do not exist")
	}

	return nil
}

// CloningRepo pulls down the specified git repo to the destination folder
func CloningRepo(dst string, src string) error {
	/* TO-DO: This function clones the specified repo to a destination folder
	 */

	fmt.Println("Cloning Repository ...")
	// clone the given repository to the given directory
	fmt.Printf("git clone %s %s --recursive", src, dst)

	r, err := git.PlainClone(dst, false, &git.CloneOptions{
		URL:               src,
		RecurseSubmodules: git.DefaultSubmoduleRecursionDepth,
	})
	if err != nil {
		fmt.Printf("\nFailed to clone repoistory to the destination folder; %s.\n", dst)
		return err
	}

	// retrieving the branch being pointed by HEAD
	ref, err := r.Head()
	if err != nil {
		return err
	}

	// retrieving the commit object
	commit, err := r.CommitObject(ref.Hash())
	if err != nil {
		return err
	}

	fmt.Println(commit)

	return nil
}
