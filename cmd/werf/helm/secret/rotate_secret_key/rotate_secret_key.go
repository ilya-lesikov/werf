package secret

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/flant/werf/cmd/werf/common"
	"github.com/flant/werf/pkg/deploy/secret"
	"github.com/flant/werf/pkg/logger"
	"github.com/flant/werf/pkg/util"
	"github.com/flant/werf/pkg/werf"
)

var CommonCmdData common.CmdData

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                   "rotate-secret-key [EXTRA_SECRET_VALUES_FILE_PATH...]",
		DisableFlagsInUseLine: true,
		Short:                 "Regenerate secret files with new secret key",
		Long: common.GetLongCommandDescription(`Regenerate secret files with new secret key.

Old key should be specified in the $WERF_OLD_SECRET_KEY.
New key should reside either in the $WERF_SECRET_KEY or .werf_secret_key file.

Command will extract data with the old key, generate new secret data and rewrite files:
* standard raw secret files in the .helm/secret folder;
* standard secret values yaml file .helm/secret-values.yaml;
* additional secret values yaml files specified with EXTRA_SECRET_VALUES_FILE_PATH params`),
		Annotations: map[string]string{
			common.CmdEnvAnno: common.EnvsDescription(common.WerfSecretKey, common.WerfOldSecretKey),
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := common.ProcessLogOptions(&CommonCmdData); err != nil {
				common.PrintHelp(cmd)
				return err
			}
			return runRotateSecretKey(cmd, args...)
		},
	}

	common.SetupDir(&CommonCmdData, cmd)
	common.SetupTmpDir(&CommonCmdData, cmd)
	common.SetupHomeDir(&CommonCmdData, cmd)

	common.SetupLogOptions(&CommonCmdData, cmd)

	return cmd
}

func runRotateSecretKey(cmd *cobra.Command, secretValuesPaths ...string) error {
	if err := werf.Init(*CommonCmdData.TmpDir, *CommonCmdData.HomeDir); err != nil {
		return fmt.Errorf("initialization error: %s", err)
	}

	projectDir, err := common.GetProjectDir(&CommonCmdData)
	if err != nil {
		return fmt.Errorf("getting project dir failed: %s", err)
	}

	newSecret, err := secret.GetManager(projectDir)
	if err != nil {
		return err
	}

	oldSecretKey := os.Getenv("WERF_OLD_SECRET_KEY")
	if oldSecretKey == "" {
		common.PrintHelp(cmd)
		return fmt.Errorf("WERF_OLD_SECRET_KEY environment required")
	}

	oldSecret, err := secret.NewManager([]byte(oldSecretKey), secret.NewManagerOptions{IgnoreWarning: true})
	if err != nil {
		return err
	}

	return secretsRegenerate(newSecret, oldSecret, projectDir, secretValuesPaths...)
}

func secretsRegenerate(newManager, oldManager secret.Manager, projectPath string, secretValuesPaths ...string) error {
	var secretFilesPaths []string
	regeneratedFilesData := map[string][]byte{}
	secretFilesData := map[string][]byte{}
	secretValuesFilesData := map[string][]byte{}

	helmChartPath := filepath.Join(projectPath, ".helm")
	isHelmChartDirExist, err := util.FileExists(helmChartPath)
	if err != nil {
		return err
	}

	if isHelmChartDirExist {
		defaultSecretValuesPath := filepath.Join(helmChartPath, "secret-values.yaml")
		isDefaultSecretValuesExist, err := util.FileExists(defaultSecretValuesPath)
		if err != nil {
			return err
		}

		if isDefaultSecretValuesExist {
			secretValuesPaths = append(secretValuesPaths, defaultSecretValuesPath)
		}

		secretDirectory := filepath.Join(helmChartPath, "secret")
		isSecretDirectoryExist, err := util.FileExists(defaultSecretValuesPath)
		if err != nil {
			return err
		}

		if isSecretDirectoryExist {
			err = filepath.Walk(secretDirectory,
				func(path string, info os.FileInfo, err error) error {
					if err != nil {
						return err
					}

					fileInfo, err := os.Stat(path)
					if err != nil {
						return err
					}

					if !fileInfo.IsDir() {
						secretFilesPaths = append(secretFilesPaths, path)
					}

					return nil
				})
			if err != nil {
				return err
			}
		}
	}

	pwd, err := os.Getwd()
	if err != nil {
		return err
	}

	secretFilesData, err = readFilesToDecode(secretFilesPaths, pwd)
	if err != nil {
		return err
	}

	secretValuesFilesData, err = readFilesToDecode(secretValuesPaths, pwd)
	if err != nil {
		return err
	}

	if err := regenerateSecrets(secretFilesData, regeneratedFilesData, oldManager.Decrypt, newManager.Encrypt); err != nil {
		return err
	}

	if err := regenerateSecrets(secretValuesFilesData, regeneratedFilesData, oldManager.DecryptYamlData, newManager.EncryptYamlData); err != nil {
		return err
	}

	for filePath, fileData := range regeneratedFilesData {
		err := logger.LogSecondaryProcess(fmt.Sprintf("Saving file '%s'", filePath), logger.LogProcessOptions{}, func() error {
			fileData = append(bytes.TrimSpace(fileData), []byte("\n")...)
			return ioutil.WriteFile(filePath, fileData, 0644)
		})

		if err != nil {
			return err
		}
	}

	return nil
}

func regenerateSecrets(filesData, regeneratedFilesData map[string][]byte, decodeFunc, encodeFunc func([]byte) ([]byte, error)) error {
	for filePath, fileData := range filesData {
		err := logger.LogSecondaryProcess(fmt.Sprintf("Regenerating file '%s'", filePath), logger.LogProcessOptions{}, func() error {
			data, err := decodeFunc(fileData)
			if err != nil {
				return fmt.Errorf("check old encryption key and file data: %s", err)
			}

			resultData, err := encodeFunc(data)
			if err != nil {
				return err
			}

			regeneratedFilesData[filePath] = resultData

			return nil
		})

		if err != nil {
			return err
		}
	}

	return nil
}

func readFilesToDecode(filePaths []string, pwd string) (map[string][]byte, error) {
	filesData := map[string][]byte{}
	for _, filePath := range filePaths {
		fileData, err := ioutil.ReadFile(filePath)
		if err != nil {
			return nil, err
		}

		if filepath.IsAbs(filePath) {
			filePath, err = filepath.Rel(pwd, filePath)
			if err != nil {
				return nil, err
			}
		}

		filesData[filePath] = bytes.TrimSpace(fileData)
	}

	return filesData, nil
}