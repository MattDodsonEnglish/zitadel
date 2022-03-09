package cmds

import (
	"errors"

	"github.com/caos/zitadel/pkg/databases"

	"github.com/caos/orbos/mntr"

	"github.com/caos/orbos/pkg/tree"

	"github.com/caos/orbos/pkg/cfg"
	"github.com/caos/orbos/pkg/git"

	"github.com/caos/orbos/pkg/kubernetes/cli"
	"github.com/caos/orbos/pkg/orb"
	"github.com/spf13/cobra"

	orbdb "github.com/caos/zitadel/operator/database/kinds/orb"
	orbzit "github.com/caos/zitadel/operator/zitadel/kinds/orb"
)

func ConfigCommand(getRv GetRootValues, ghClientID, ghClientSecret string) *cobra.Command {

	var (
		newMasterKey string
		newRepoURL   string
		newRepoKey   string
		cmd          = &cobra.Command{
			Use:     "configure",
			Short:   "Configures and reconfigures an orb",
			Long:    "Generates missing secrets where it makes sense",
			Aliases: []string{"reconfigure", "config", "reconfig"},
		}
	)

	flags := cmd.Flags()
	flags.StringVar(&newMasterKey, "masterkey", "", "Reencrypts all secrets")
	flags.StringVar(&newRepoURL, "repourl", "", "Configures the repository URL")
	flags.StringVar(&newRepoKey, "repokey", "", "Configures the used key to communicate with the repository")

	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {

		rv := getRv("configure", map[string]interface{}{
			"masterkey":  newMasterKey != "",
			"newRepoURL": newRepoURL,
			"newRepoKey": newRepoKey != "",
		}, "")
		defer func() {
			err = rv.ErrFunc(err)
		}()

		if !rv.Gitops {
			return mntr.ToUserError(errors.New("configure command is only supported with the --gitops flag"))
		}

		if err := orb.Reconfigure(
			rv.Ctx,
			rv.Monitor,
			rv.OrbConfig,
			newRepoURL,
			newMasterKey,
			newRepoKey,
			rv.GitClient,
			ghClientID,
			ghClientSecret,
		); err != nil {
			return err
		}

		k8sClient, err := cli.Client(rv.Monitor, rv.OrbConfig, rv.GitClient, rv.Kubeconfig, rv.Gitops, false)
		if err != nil {
			rv.Monitor.WithField("reason", err.Error()).Info("Continuing without having a Kubernetes connection")
			err = nil
		}

		dbClient, err := databases.NewConnection(rv.Monitor, k8sClient, rv.Gitops, rv.OrbConfig)
		if err != nil {
			return err
		}

		if err := cfg.ApplyOrbconfigSecret(
			rv.OrbConfig,
			k8sClient,
			rv.Monitor,
		); err != nil {
			return err
		}

		queried := make(map[string]interface{})
		if err := cfg.ConfigureOperators(
			rv.GitClient,
			rv.OrbConfig.Masterkey,
			append(cfg.ORBOSConfigurers(
				rv.Ctx,
				rv.Monitor,
				rv.OrbConfig,
				rv.GitClient,
			), cfg.OperatorConfigurer(
				git.DatabaseFile,
				rv.Monitor,
				rv.GitClient,
				func() (*tree.Tree, interface{}, error) {
					desired, err := rv.GitClient.ReadTree(git.DatabaseFile)
					if err != nil {
						return nil, nil, err
					}

					_, _, configure, _, _, _, err := orbdb.AdaptFunc("", nil, rv.Gitops)(rv.Monitor, desired, &tree.Tree{})
					if err != nil {
						return nil, nil, err
					}
					return desired, desired.Parsed, configure(k8sClient, queried, rv.Gitops)
				},
			), cfg.OperatorConfigurer(
				git.ZitadelFile,
				rv.Monitor,
				rv.GitClient,
				func() (*tree.Tree, interface{}, error) {
					desired, err := rv.GitClient.ReadTree(git.ZitadelFile)
					if err != nil {
						return nil, nil, err
					}

					_, _, configure, _, _, _, err := orbzit.AdaptFunc(
						"configure",
						nil,
						rv.Gitops,
						nil,
						dbClient,
					)(rv.Monitor, desired, &tree.Tree{})
					if err != nil {
						return nil, nil, err
					}
					return desired, desired.Parsed, configure(k8sClient, queried, rv.Gitops)
				},
			))); err != nil {
			return err
		}
		rv.Monitor.Info("Configuration succeeded")
		return nil
	}
	return cmd
}
