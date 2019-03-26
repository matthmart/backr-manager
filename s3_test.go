package manager

// func getSampleFiles() S3FilesByDateAsc {
// 	return S3FilesByDateAsc{
// 		S3File{Filename: "file1", Date: time.Date(2018, 12, 1, 8, 0, 0, 0, time.Local)},
// 		S3File{Filename: "file2", Date: time.Date(2018, 12, 2, 8, 0, 0, 0, time.Local)},
// 		S3File{Filename: "file3", Date: time.Date(2018, 11, 18, 8, 0, 0, 0, time.Local)},
// 	}
// }

// func TestS3FilesByDateAscSort(t *testing.T) {

// 	files := getSampleFiles()

// 	files.Sort()

// 	expected := []string{"file3", "file1", "file2"}
// 	for i, f := range expected {
// 		if files[i].Filename != f {
// 			t.Errorf("sorting failed: expected: %s got: %s", f, files[i].Filename)
// 			break
// 		}
// 	}
// }

// func TestS3FilesByDateAscLatest(t *testing.T) {

// 	files := getSampleFiles()

// 	f := files.Latest()

// 	expected := "file2"
// 	if f.Filename != expected {
// 		t.Errorf("sorting failed: expected: %s got: %s", expected, f.Filename)
// 	}
// }
