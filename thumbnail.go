package go_vlc_thumbnail

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"time"
)

// TempWorkDir is the location where VLC will temporarily save the snapshot to. Default is the process work dir (PWD).
var TempWorkDir string

// CVLCBinPath holds the location of the CVLC executable binary.
var CVLCBinPath string

// DisableHardwareAudioVideoCodec forces the CPU to do encoding, this may solve problems with
// nvidia drivers on Linux.
var DisableHardwareAudioVideoCodec = true

type pictureOutputFormat int

const (
	FORMAT_JPEG = pictureOutputFormat(iota)
	FORMAT_PNG
	FORMAT_TIFF
)

// Video structure defines which file (Source) to create a snapshot of.
// Valid OutputFormat arguments are FORMAT_JPEG, FORMAT_PNG, FORMAT_TIFF.
// The Time defines which second of the given Source a snapshot should be made.
type Video struct {
	Source       string              // Path to the video file
	OutputFormat pictureOutputFormat // The output format of the thumbnail, default is FORMAT_JPEG
	Time         int                 // Which second of the video a snapshot is to be created of

	stdOut bytes.Buffer
	stdErr bytes.Buffer
	prefix string
}

// CommandLog returns the stdout and stderr output of the exec.Command("cvlc", ...). This can be
// useful for debugging any errors not caught by this package.
// Will only contain logs after calling Generate() or GenerateTo().
func (tf Video) CommandLog() (stdout, stderr bytes.Buffer) {
	return tf.stdOut, tf.stdErr
}

// GenerateTo generates a snapshot in the given OutputFormat from the Source and saves it to the given saveLocation if the returned error is nil.
// No file extension (eg .jpg) will be added to the saveLocation path. It's up to the user
// to do this.
func (tf *Video) GenerateTo(saveLocation string) error {
	fb, err := tf.Generate()
	if err != nil {
		return err
	}

	// Write to file
	return ioutil.WriteFile(saveLocation, fb, 0644)
}

// Generate returns a snapshot in the given OutputFormat from the Source in a byte slice or an error.
func (tf *Video) Generate() (file []byte, err error) {
	err = tf.checkInputErrors()
	if err != nil {
		return
	}

	// Create file stamp
	tn := time.Now().UnixMilli()
	tf.prefix = "vlc_conv_" + strconv.Itoa(int(tn))

	// Run VLC
	err = tf.run()
	if err != nil {
		return nil, err
	}

	// Read the file
	file, err = ioutil.ReadFile(tf.tempFileLocation())
	if err != nil {
		return nil, err
	}

	err = tf.cleanTempFile()
	if err != nil {
		return nil, err
	}

	return file, nil
}

// Checks for any user input errors.
func (tf *Video) checkInputErrors() (err error) {
	// Check if the work dir is set, if not use the PWD
	if TempWorkDir == "" {
		TempWorkDir, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("TEMP_WORK_DIR was not set, os.Getwd() failed: %s", err)
		}
	}

	if tf.OutputFormat < FORMAT_JPEG || tf.OutputFormat > FORMAT_TIFF {
		return fmt.Errorf("value %d is invalid for (*Video).OutputFormat", tf.OutputFormat)
	}

	if tf.Time < 0 {
		return fmt.Errorf("invalid value for (*Video).Time, must be equal to 0 or greater than 0")
	}

	// Check if file exists
	_, err = os.Stat(tf.Source)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("no source file found at location %s (*Video).Source", tf.Source)
		} else {
			return err
		}
	}

	// Check if vlc binary exists
	if CVLCBinPath == "" {
		// Find vlc in bin
		CVLCBinPath, err = findVlc()
		if err != nil {
			return fmt.Errorf("VLC_BIN_PATH was not set, failed to find vlc location: %s", err)
		}
	} else {
		_, err = os.Stat(CVLCBinPath)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return fmt.Errorf("VLC_BIN_PATH was set but not found at: %s", CVLCBinPath)
			} else {
				return err
			}
		}
	}

	return nil
}

// Returns the workdir for the temporary files. By default, this is the process work dir (pwd).
func (tf Video) workdir() (wdir string) {
	wdir = TempWorkDir

	// the vlc flag --scene-path requires a trailing slash
	if !strings.HasSuffix(wdir, "/") {
		wdir += "/"
	}
	return wdir
}

// Returns the suffix as a string defined by OutputFormat.
func (tf Video) suffix() (output string) {
	switch tf.OutputFormat {
	case FORMAT_JPEG:
		output = "jpg"
	case FORMAT_PNG:
		output = "png"
	case FORMAT_TIFF:
		output = "tiff"
	}
	return
}

// Returns the location of the temporary file that VLC generates.
func (tf *Video) tempFileLocation() string {
	return tf.workdir() + tf.prefix + "." + tf.suffix()
}

// Removes the generated file by VLC
func (tf *Video) cleanTempFile() (err error) {
	err = os.Remove(tf.tempFileLocation())
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("failed to remove temp file %s , reason: %s", tf.tempFileLocation(), err)
		}
	}
	return nil
}
