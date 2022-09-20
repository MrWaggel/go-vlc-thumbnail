package go_vlc_thumbnail

import (
	"bufio"
	"bytes"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// Create the args for the cvlc.
func (tf *Video) generateArguments() []string {
	args := make([]string, 0)
	args = append(args, "file://"+tf.Source+"")

	if DisableHardwareAudioVideoCodec {
		args = append(args, "--avcodec-hw")
		args = append(args, "none")
	}

	args = append(args, "--rate=99999")
	args = append(args, "--video-filter=scene")
	args = append(args, "--vout=dummy")
	args = append(args, "--aout=dummy")
	args = append(args, "--start-time="+strconv.Itoa(tf.Time))
	args = append(args, "--stop-time="+strconv.Itoa(tf.Time+1))
	args = append(args, "--scene-format="+tf.suffix())
	args = append(args, "--scene-prefix="+tf.prefix)
	args = append(args, "--scene-replace")
	args = append(args, "--scene-path="+tf.workdir())
	args = append(args, "vlc://quit")

	return args
}

// Runs the cvlc command to generate the actual snapshot.
func (tf *Video) run() (err error) {
	cmd := exec.Command(CVLCBinPath, tf.generateArguments()...)

	cmd.Stdout = &tf.stdOut
	cmd.Stderr = &tf.stdErr

	err = cmd.Run()
	if err != nil {
		e2 := checkVlcErrors(&tf.stdErr)
		if e2 != nil {
			return e2
		}
		return fmt.Errorf("failed to start VLC reason (see (*ThumnailFile).CommandLog(): %s", err)
	}

	// Check for VLC errors, vlc doesn't always return a status code which is not equal to 0
	// if an error occurs.
	return checkVlcErrors(&tf.stdOut)
}

func checkVlcErrors(buf *bytes.Buffer) (err error) {

	scanner := bufio.NewScanner(buf)
	for scanner.Scan() {
		err = checkVlcError(scanner.Text())
		if err != nil {
			return fmt.Errorf("vlc error: %s", err)
		}
	}
	return nil
}

func checkVlcError(line string) (err error) {
	if strings.Contains(line, "could not create snapshot") {
		return fmt.Errorf(line)
	}
	if strings.Contains(line, "filesystem stream error") {
		return fmt.Errorf(line)
	}
	if strings.Contains(line, "could not identify codec") {
		return fmt.Errorf(line)
	}
	return nil
}

// Helper to find cvlc on the host. Only works if cvlc is present in one of the
// system's bin locations (eg /usr/bin)
func findVlc() (location string, err error) {
	c := exec.Command("/usr/bin/which", "cvlc")
	buf := bytes.Buffer{}
	c.Stdout = &buf
	err = c.Run()
	if err != nil {
		return "", err
	}

	if c.ProcessState.ExitCode() != 0 {
		return "", fmt.Errorf("failed to find VLC executable")
	}

	// Don't forget to remove the newline!
	return strings.ReplaceAll(buf.String(), "\n", ""), nil
}
