package vvpk

import (
	"fmt"
	"path/filepath"
	"testing"
)

func TestOpenAllVpk(t *testing.T) {

	var workshopDir = "/media/lucas/VolumeD/apps/steam/steamapps/common/Left 4 Dead 2/left4dead2/addons/workshop"

	pattern := filepath.Join(workshopDir, "*.vpk")

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
	var modPath = "/media/lucas/VolumeD/apps/steam/steamapps/common/Left 4 Dead 2/left4dead2/addons/workshop/121086524.vpk"

	vpkv1, err := OpenVpk(modPath)

	if err != nil {
		panic(err)
	}

	fmt.Println(vpkv1.Version)
	// for idx, item := range vpkv1.Entries {
	// 	fmt.Printf("%d: %s ---> %s --> %s\n", idx, item.Extension, item.Path, item.Filename)
	// }
}

func TestVerifyBoundary(t *testing.T) {
	// var modPath = "/media/lucas/VolumeD/apps/steam/steamapps/common/Left 4 Dead 2/left4dead2/addons/workshop/121796400.vpk"
	var modPath = "/media/lucas/VolumeD/apps/steam/steamapps/common/Left 4 Dead 2/left4dead2/addons/workshop/121086524.vpk"

	finish, err := VerifyBoundary(modPath)

	if err != nil {
		panic(err)
	}

	fmt.Println(finish)
}
