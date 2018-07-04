package uuid_test

import (
	"fmt"
	"net/url"
	"time"
	"github.com/myesui/uuid"
	"github.com/myesui/uuid/savers"
)

func Example() {

	saver := new(savers.FileSystemSaver)
	saver.Report = true
	saver.Duration = time.Second * 3

	// Run before any v1 or v2 UUIDs to ensure the savers takes
	uuid.RegisterSaver(saver)

	id, _ := uuid.Parse("6ba7b810-9dad-11d1-80b4-00c04fd430c8")
	fmt.Printf("version %d variant %x: %s\n", id.Version(), id.Variant(), id)

	uuid.New(id.Bytes())

	id1 := uuid.NewV1()
	fmt.Printf("version %d variant %x: %s\n", id1.Version(), id1.Variant(), id1)

	id4 := uuid.NewV4()
	fmt.Printf("version %d variant %x: %s\n", id4.Version(), id4.Variant(), id4)

	id3 := uuid.NewV3(id1, id4)

	url, _ := url.Parse("www.example.com")

	id5 := uuid.NewV5(uuid.NameSpaceURL, url)

	if uuid.Equal(id1, id3) {
		fmt.Println("Will never happen")
	}

	if uuid.Compare(uuid.NameSpaceDNS, uuid.NameSpaceDNS) == 0 {
		fmt.Println("They are equal")
	}

	// Default Format is Canonical
	fmt.Println(uuid.Formatter(id5, uuid.FormatCanonicalCurly))

	uuid.SwitchFormat(uuid.FormatCanonicalBracket)
}

func ExampleNewV1() {
	id1 := uuid.NewV1()
	fmt.Printf("version %d variant %d: %s\n", id1.Version(), id1.Variant(), id1)
}

func ExampleNewV2() {
	id2 := uuid.NewV2(uuid.SystemIdUser)
	fmt.Printf("version %d variant %d: %s\n", id2.Version(), id2.Variant(), id2)
}

func ExampleNewV3() {
	id, _ := uuid.Parse("6ba7b810-9dad-11d1-80b4-00c04fd430c8")
	id3 := uuid.NewV3(id, "test")
	fmt.Printf("version %d variant %x: %s\n", id3.Version(), id3.Variant(), id3)
}

func ExampleNewV4() {
	id := uuid.NewV4()
	fmt.Printf("version %d variant %x: %s\n", id.Version(), id.Variant(), id)
}

func ExampleNewV5() {
	id5 := uuid.NewV5(uuid.NameSpaceURL, "test")
	fmt.Printf("version %d variant %x: %s\n", id5.Version(), id5.Variant(), id5)
}

func ExampleParse() {
	id, err := uuid.Parse("6ba7b810-9dad-11d1-80b4-00c04fd430c8")
	if err != nil {
		fmt.Println("error:", err)
	}
	fmt.Println(id)
}

func ExampleRegisterSaver() {
	saver := new(savers.FileSystemSaver)
	saver.Report = true
	saver.Duration = 3 * time.Second

	// Run before any v1 or v2 UUIDs to ensure the savers takes
	uuid.RegisterSaver(saver)
	id1 := uuid.NewV1()
	fmt.Printf("version %d variant %x: %s\n", id1.Version(), id1.Variant(), id1)
}

func ExampleFormatter() {
	id4 := uuid.NewV4()
	fmt.Println(uuid.Formatter(id4, uuid.FormatCanonicalCurly))
}

func ExampleSwitchFormat() {
	uuid.SwitchFormat(uuid.FormatCanonicalBracket)
	u4 := uuid.NewV4()
	fmt.Printf("version %d variant %x: %s\n", u4.Version(), u4.Variant(), u4)
}
