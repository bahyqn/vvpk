package vvpk

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"testing"
)

const (
	DIR         = "/media/lucas/VolumeD/apps/steam/steamapps/common/Left 4 Dead 2/left4dead2"
	WORKSHOPDIR = "/media/lucas/VolumeD/apps/steam/steamapps/common/Left 4 Dead 2/left4dead2/addons/workshop"
	// urban flight 121086524.vpk
	modPath1 = "/media/lucas/VolumeD/apps/steam/steamapps/common/Left 4 Dead 2/left4dead2/addons/workshop/121086524.vpk"
	// warcalona 1
	modPath2 = "/media/lucas/VolumeD/apps/steam/steamapps/common/Left 4 Dead 2/left4dead2/addons/workshop/121796400.vpk"
	// few files
	modPath3 = "/media/lucas/VolumeD/apps/steam/steamapps/common/Left 4 Dead 2/left4dead2/addons/workshop/3646935257.vpk"
	modPath4 = "/media/lucas/VolumeD/apps/steam/steamapps/common/Left 4 Dead 2/left4dead2/addons/workshop/397962151.vpk"
	//
	modPath5 = "/media/lucas/VolumeD/apps/steam/steamapps/common/Left 4 Dead 2/left4dead2/addons/workshop/128424524.vpk"
	// super healing
	modPath6 = "/media/lucas/VolumeD/apps/steam/steamapps/common/Left 4 Dead 2/left4dead2/addons/workshop/1846565331.vpk"
)

func TestOpenAllVpk(t *testing.T) {

	pattern := filepath.Join(WORKSHOPDIR, "*.vpk")

	tvpks, err := filepath.Glob(pattern)
	if err != nil {
		panic(err)
	}

	for fileIdx, item := range tvpks {
		_, err := OpenVpkDev(item)

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
	// fmap := OpenVpk(modPath1)
	// testMods := []string{modPath1, modPath2, modPath3, modPath4, modPath5}
	testMods := []string{modPath6}

	for idx, el := range testMods {
		fmt.Println("---------------", idx, "-------------")
		fmap := OpenVpk(el)

		fmt.Println(fmap)

		// addoninfo, err := StringToMap(fmap["addoninfo.txt"])
		// if err != nil {
		// 	panic("x1")
		// }
		// fmt.Printf("%v\n\n", addoninfo)

		// tmap, err := StringToMap(fmap["missions"])
		// if err != nil {
		// 	panic("x1")
		// }
		// fmt.Printf("%v\n", tmap)
	}
}

func TestOpenVpkDev(t *testing.T) {

	vpkv1, err := OpenVpkDev(modPath6)

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

func TestReadTxtFile(t *testing.T) {
	vpkv1, err := OpenVpkDev(modPath3)

	if err != nil {
		panic(err)
	}

	f, err := os.Open(modPath1)

	if err != nil {
		panic("xx111")
	}
	defer f.Close()

	fmt.Println(vpkv1.Version)
	// for idx, item := range vpkv1.Entries {
	// 	fmt.Printf("%d: %s ---> %s --> %s\n", idx, item.Extension, item.Path, item.Filename)

	// 	if item.Extension == "txt" && item.Filename == "addoninfo" {
	// 		fmt.Println(item.Extension)
	// 		fmt.Println(item.Path)
	// 		fmt.Println(item.Filename)
	// 		fmt.Println(item.ArchiveIndex)
	// 		fmt.Println(item.Preload)
	// 		fmt.Println(item.EntryOffset)
	// 		fmt.Println(item.EntryLength)

	// 		// ReadTxtFile(f, &item)
	// 	}
	// }
}

func TestOpenFile(t *testing.T) {
	content, _ := OpenAddonlist(path.Join(DIR, "addonlist.txt"))

	for _, el := range content {
		// fmt.Printf("%d ---> %s \n", idx, el)
		fmt.Println(el)
	}

	UpdateModStatus(content, "1928446407", "0")
	for _, el := range content {
		fmt.Println(el)
	}
}
