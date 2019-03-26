package manager

// // S3Client defines methods to communicate
// // with the S3 API
// type S3Client interface {
// 	Fetch() (S3FilesByFolder, error)
// 	Delete(files []S3File) error
// }

// // S3FilesByFolder stores S3 files, grouped by their folder
// type S3FilesByFolder map[string]S3FilesByDateAsc

// // S3File represents a file stored in object storage
// type S3File struct {
// 	Filename string
// 	Date     time.Time
// 	Size     int64
// }

// // S3FilesByDateAsc stores a slice of S3 files, which should be sorted by date,
// // from older to earlier
// type S3FilesByDateAsc []S3File

// func (files S3FilesByDateAsc) Len() int {
// 	return len(files)
// }

// func (files S3FilesByDateAsc) Less(i, j int) bool {
// 	return files[i].Date.Before(files[j].Date)
// }

// func (files S3FilesByDateAsc) Swap(i, j int) {
// 	files[i], files[j] = files[j], files[i]
// }

// // Sort sorts the files by date (asc)
// func (files S3FilesByDateAsc) Sort() {
// 	sort.Sort(files)
// }

// // Latest returns the most recent file from the list
// func (files S3FilesByDateAsc) Latest() S3File {
// 	// sort the backups
// 	files.Sort()
// 	// return the latest element
// 	s := []S3File(files)
// 	return s[len(s)-1]
// }
