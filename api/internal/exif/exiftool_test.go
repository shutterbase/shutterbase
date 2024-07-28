package exif

/*
import (
	"context"
	"os"
	"testing"

	"github.com/shutterbase/shutterbase/internal/client"
	"github.com/stretchr/testify/assert"
)

func TestImageTagging(t *testing.T) {
	c := client.NewClient("http://localhost:8090")
	assert.NotNil(t, c)

	ctx := context.Background()

	err := c.Login(ctx, "test.user@shutterbase.io", "test1234")
	assert.Nil(t, err)
	assert.NotEmpty(t, c.Auth.Token)

	imageFile, err := GetImageFileWithAdjustedExifData(ctx, "dzjxowsxmsa8e09", c)
	assert.Nil(t, err)
	assert.NotNil(t, imageFile)

	// write to file
	os.WriteFile("test.jpg", imageFile, 0644)
}
*/
