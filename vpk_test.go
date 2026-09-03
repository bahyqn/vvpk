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

	for idx, item := range tvpks {
		vpkv1, err := OpenVpk(item)

		if err != nil {
			t.Fatalf("open %q failed: %v", item, err)
		}

		len := vpkv1.LengthValidate()
		fmt.Printf("%d -> len: %d \n", idx, len)
	}
}

func TestOpenVpk(t *testing.T) {
	var modPath = "/media/lucas/VolumeD/apps/steam/steamapps/common/Left 4 Dead 2/left4dead2/addons/workshop/121086524.vpk"

	vpkv1, err := OpenVpk(modPath)

	if err != nil {
		panic("xxxxxxxxxx")
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
		fmt.Println(err)
	}

	fmt.Println(finish)
}
