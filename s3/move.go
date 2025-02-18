package s3

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/pkg/errors"
	"github.com/viant/afs/option"
	"github.com/viant/afs/storage"
	"path"
	"strings"
)

func (s *storager) Move(ctx context.Context, sourcePath, destBucket, destPath string, options ...storage.Option) error {
	sourcePath = strings.Trim(sourcePath, "/")
	destPath = strings.Trim(destPath, "/")
	source, err := s.get(ctx, sourcePath, options)
	if isNotFound(err) {
		objectOpt := &option.ObjectKind{}
		if _, ok := option.Assign(options, &objectOpt); ok && objectOpt.File {
			return err
		}
		infoList, err := s.List(ctx, sourcePath)
		if err != nil {
			return err
		}
		if len(infoList) == 0 {
			return fmt.Errorf("%v: not found", sourcePath)
		}
		for i := 1; i < len(infoList); i++ {
			name := infoList[i].Name()
			if err = s.Move(ctx, path.Join(sourcePath, name), destBucket, path.Join(destPath, name)); err != nil {
				return err
			}
		}
		return nil
	}
	if err != nil {
		return err
	}
	copyInput := &s3.CopyObjectInput{
		CopySource: aws.String(s.bucket + "/" + sourcePath),
		Key:        &destPath,
		Bucket:     &destBucket,
	}
	if source.Size() >= maxCopySize {
		copyer := newCopyer(s.S3, source, defaultPartSize, copyInput)
		err = copyer.copy(ctx)
	} else {
		_, err = s.S3.CopyObjectWithContext(ctx, copyInput)
	}
	if err != nil {
		err = errors.Wrapf(err, "failed to move: s3://%v/%v to s3://%v/%v", s.bucket, sourcePath, destBucket, destPath)
	}
	if err == nil {
		err = s.Delete(ctx, sourcePath)
	}
	return err
}
