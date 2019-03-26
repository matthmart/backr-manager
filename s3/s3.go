package s3

// var sharedClient *client

// func NewFileRepository(config manager.S3Config) (manager.FileRepository, error) {
// 	minioClient, err := minio.New(config.Endpoint, config.AccessKey, config.SecretKey, config.UseTLS)
// 	if err != nil {
// 		return nil, err
// 	}

// 	sharedClient = client{
// 		bucket:      config.Bucket,
// 		minioClient: minioClient,
// 	}

// 	return &sharedClient, nil
// }

// // NewClient returns a S3 client
// func newClient(config manager.S3Config) (manager.S3Client, error) {
// 	minioClient, err := minio.New(config.Endpoint, config.AccessKey, config.SecretKey, config.UseTLS)
// 	if err != nil {
// 		return nil, err
// 	}

// 	sharedClient = client{
// 		bucket:      config.Bucket,
// 		minioClient: minioClient,
// 	}

// 	return &sharedClient, nil
// }

// type client struct {
// 	bucket      string
// 	minioClient *minio.Client
// }

// func (client *client) Fetch() (manager.S3FilesByFolder, error) {

// 	// Create a done channel.
// 	doneCh := make(chan struct{})
// 	defer close(doneCh)

// 	// Recursively list all objects
// 	recursive := true

// 	filesByFolder := manager.S3FilesByFolder{}
// 	for object := range client.minioClient.ListObjectsV2(client.bucket, "", recursive, doneCh) {
// 		components := strings.Split(object.Key, "/")

// 		// ensure that there is no more than 2 levels (bucket/folder/files)
// 		if len(components) == 2 {
// 			folder := components[0]
// 			// init files slice if needed
// 			if _, ok := filesByFolder[folder]; !ok {
// 				filesByFolder[folder] = []manager.S3File{}
// 			}

// 			// append the S3 file
// 			filesByFolder[folder] = append(filesByFolder[folder], manager.S3File{
// 				Filename: components[1],
// 				Date:     object.LastModified,
// 				Size:     object.Size,
// 			})
// 		}
// 	}

// 	// sort files
// 	for folder := range filesByFolder {
// 		sort.Sort(filesByFolder[folder])
// 	}

// 	return filesByFolder, nil
// }

// func (client *client) Delete(files []manager.S3File) error {

// 	return nil
// }
