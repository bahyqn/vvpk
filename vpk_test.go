package vvpk

import (
	"fmt"
	"path/filepath"
	"testing"
)

const (
	WORKSHOPDIR = "/media/lucas/VolumeD/apps/steam/steamapps/common/Left 4 Dead 2/left4dead2/addons/workshop"
	modPath1    = "/media/lucas/VolumeD/apps/steam/steamapps/common/Left 4 Dead 2/left4dead2/addons/workshop/121086524.vpk"
	modPath2    = "/media/lucas/VolumeD/apps/steam/steamapps/common/Left 4 Dead 2/left4dead2/addons/workshop/121796400.vpk"
	modPath3    = "/media/lucas/VolumeD/apps/steam/steamapps/common/Left 4 Dead 2/left4dead2/addons/workshop/3646935257.vpk"
	modPath4    = "/media/lucas/VolumeD/apps/steam/steamapps/common/Left 4 Dead 2/left4dead2/addons/workshop/397962151.vpk"
)

func TestOpenAllVpk(t *testing.T) {

	pattern := filepath.Join(WORKSHOPDIR, "*.vpk")

	tvpks, err := filepath.Glob(pattern)
	if err != nil {
		panic(err)
	}

	for fileIdx, item := range tvpks {
		_, err := OpenVpk(item)

		if err != nil {
			t.Fatalf("open %q failed: %v", item, err)
		}

		fmt.Printf("%d ---> %s\n", fileIdx, item)

		// for idx, item := range vpkv1.Entries {
		// 	fmt.Printf("%d: %s ---> %s --> %s\n", idx, item.Extension, item.Path, item.Filename)
		// }

		// fmt.Println()
	}
}

func TestOpenVpk(t *testing.T) {

	vpkv1, err := OpenVpk(modPath1)

	if err != nil {
		panic(err)
	}

	fmt.Println(vpkv1.Version)
	for idx, item := range vpkv1.Entries {
		fmt.Printf("%d: %s ---> %s --> %s\n", idx, item.Extension, item.Path, item.Filename)
	}
}

func TestVerifyBoundary(t *testing.T) {
	ok, err := VerifyBoundary(modPath1)

	if err != nil {
		panic(err)
	}

	fmt.Println(ok)
}

func TestVerifyChecksum(t *testing.T) {

	failedEntries, err := VerifyChecksums(modPath4)

	if err != nil {
		panic(err)
	}

	fmt.Println(failedEntries, err)
}

func TestVerifyChecksums(t *testing.T) {
	pattern := filepath.Join(WORKSHOPDIR, "*.vpk")
	tvpks, err := filepath.Glob(pattern)
	if err != nil {
		panic(err)
	}

	for fileIdx, path := range tvpks {
		failedEntries, err := VerifyChecksums(path)

		if err != nil {
			panic(err)
		}

		fmt.Println(failedEntries)
		fmt.Printf("%d \t %s \n\n", fileIdx, err)

		// for idx, item := range vpkv1.Entries {
		// 	fmt.Printf("%d: %s ---> %s --> %s\n", idx, item.Extension, item.Path, item.Filename)
		// }

		// fmt.Println()
	}
}
