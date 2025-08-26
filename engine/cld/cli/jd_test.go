package cli

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"
)

func TestNewJDCmds_Structure(t *testing.T) {
	t.Parallel()
	c := Commands{}
	domain := domain.NewDomain("/tmp", "foo")
	root := c.NewJDCmds(domain)

	// root
	require.Equal(t, "jd", root.Use)
	require.Equal(t, "Manage job distributor interactions", root.Short)

	// persistent flag
	f := root.PersistentFlags().Lookup("environment")
	require.NotNil(t, f)
	require.Equal(t, "e", f.Shorthand)
	require.NotNil(t, root.PersistentFlags().Lookup("environment"))

	// subcommands: node & job
	subs := root.Commands()
	require.Len(t, subs, 2)
	require.ElementsMatch(t,
		[]string{"node", "job"},
		[]string{subs[0].Use, subs[1].Use},
	)
}

func TestJDNodeList_Metadata(t *testing.T) {
	t.Parallel()
	cmd := (&Commands{}).newJDNodeList(domain.NewDomain("", "foo"))

	require.Equal(t, "list", cmd.Use)
	require.Equal(t, []string{"ls"}, cmd.Aliases)
	require.Equal(t, jdNodeListLong, cmd.Long)
	require.Equal(t, jdNodeListExample, cmd.Example)
	require.Equal(t, "List out nodes registered with job distributor", cmd.Short)

	// flags
	require.NotNil(t, cmd.Flags().Lookup("label"))
	require.Equal(t, "l", cmd.Flags().Lookup("label").Shorthand)

	require.NotNil(t, cmd.Flags().Lookup("dons"))
	require.Empty(t, cmd.Flags().Lookup("dons").Shorthand)

	require.NotNil(t, cmd.Flags().Lookup("view-jobs"))
	require.Empty(t, cmd.Flags().Lookup("view-jobs").Shorthand)

	require.NotNil(t, cmd.Flags().Lookup("format"))
	require.Equal(t, "f", cmd.Flags().Lookup("format").Shorthand)
}

func TestJDNodeInspect_Metadata(t *testing.T) {
	t.Parallel()
	cmd := (&Commands{}).newJDNodeInspect(domain.NewDomain("", "foo"))

	require.Equal(t, "inspect", cmd.Use)
	require.Equal(t, []string{"i"}, cmd.Aliases)
	require.Equal(t, jdNodeInspectLong, cmd.Long)
	require.Equal(t, jdNodeInspectExample, cmd.Example)
	require.Equal(t, "Inspect chain configs for chainlink node(s) with job-distributor", cmd.Short)

	require.NotNil(t, cmd.Flags().Lookup("label"))
	require.Equal(t, "l", cmd.Flags().Lookup("label").Shorthand)

	require.NotNil(t, cmd.Flags().Lookup("format"))
	require.Equal(t, "f", cmd.Flags().Lookup("format").Shorthand)
}

func TestJDNodePatchLabels_Metadata(t *testing.T) {
	t.Parallel()
	cmd := (&Commands{}).newJDNodePatchLabels(domain.NewDomain("", "foo"))

	require.Equal(t, "labels-patch", cmd.Use)
	require.Equal(t, jdNodePatchLong, cmd.Long)
	require.Equal(t, jdNodePatchExample, cmd.Example)
	require.Equal(t, "Patch labels for nodes in job-distributor", cmd.Short)

	require.NotNil(t, cmd.Flags().Lookup("label"))
	require.Equal(t, "l", cmd.Flags().Lookup("label").Shorthand)

	require.NotNil(t, cmd.Flags().Lookup("dry-run"))
	require.Equal(t, "d", cmd.Flags().Lookup("dry-run").Shorthand)
}

func TestJDNodeRegister_Metadata(t *testing.T) {
	t.Parallel()
	cmd := (&Commands{}).newJDNodeRegister(domain.NewDomain("", "foo"))

	require.Equal(t, "register", cmd.Use)
	require.Equal(t, jdNodeRegisterLong, cmd.Long)
	require.Equal(t, jdNodeRegisterExample, cmd.Example)
	require.Equal(t, "Register chainlink node with job-distributor", cmd.Short)

	name := cmd.Flags().Lookup("name")
	require.NotNil(t, name)
	require.Equal(t, "n", name.Shorthand)

	csa := cmd.Flags().Lookup("csa-address")
	require.NotNil(t, csa)
	require.Equal(t, "a", csa.Shorthand)

	// missing required flags should error
	cmd.SetArgs([]string{})
	err := cmd.Execute()
	require.Error(t, err)
	require.Contains(t, err.Error(), `required flag(s) "csa-address", "name" not set`)

	// optional flags exist
	require.NotNil(t, cmd.Flags().Lookup("bootstrap"))
	require.NotNil(t, cmd.Flags().Lookup("label"))
}

func TestJDNodeBatchRegister_Metadata(t *testing.T) {
	t.Parallel()
	cmd := (&Commands{}).newJDNodeBatchRegister(domain.NewDomain("", "foo"))

	require.Equal(t, "batch-register", cmd.Use)
	require.Equal(t, jdNodeBatchRegisterLong, cmd.Long)
	require.Equal(t, jdNodeBatchRegisterExample, cmd.Example)
	require.Equal(t, "Register chainlink nodes with job-distributor", cmd.Short)

	config := cmd.Flags().Lookup("config")
	require.NotNil(t, config)
	require.Equal(t, "d", config.Shorthand)

	// missing required flag should error
	cmd.SetArgs([]string{})
	err := cmd.Execute()
	require.Error(t, err)
	require.Contains(t, err.Error(), `required flag(s) "config" not set`)

	require.NotNil(t, cmd.Flags().Lookup("label"))
}

func TestJDJobPropose_Metadata(t *testing.T) {
	t.Parallel()
	cmd := (&Commands{}).newJDJobPropose(domain.NewDomain("", "foo"))

	require.Equal(t, "propose", cmd.Use)
	require.Equal(t, jdJobProposeLong, cmd.Long)
	require.Equal(t, jdJobProposeExample, cmd.Example)
	require.Equal(t, "Propose a single job to multiple nodes", cmd.Short)

	j := cmd.Flags().Lookup("jobspec")
	require.NotNil(t, j)
	require.Equal(t, "j", j.Shorthand)

	// missing required flag should error
	cmd.SetArgs([]string{})
	err := cmd.Execute()
	require.Error(t, err)
	require.Contains(t, err.Error(), `required flag(s) "jobspec" not set`)

	require.NotNil(t, cmd.Flags().Lookup("label"))
}

func TestJDJobBatchPropose_Metadata(t *testing.T) {
	t.Parallel()
	cmd := (&Commands{}).newJDJobBatchPropose(domain.NewDomain("", "foo"))

	require.Equal(t, "batch-propose", cmd.Use)
	require.Equal(t, jdJobBatchProposeLong, cmd.Long)
	require.Equal(t, jdJobBatchProposeExample, cmd.Example)
	require.Equal(t, "Propose all jobs in a jobspecs artifact to nodes", cmd.Short)

	require.NotNil(t, cmd.Flags().Lookup("jobspec"))
	require.NotNil(t, cmd.Flags().Lookup("jobs"))
}

func TestJDNodeSaveAll_Metadata(t *testing.T) {
	t.Parallel()
	cmd := (&Commands{}).newJDNodeSaveAll(domain.NewDomain("", "foo"))

	require.Equal(t, "save-all", cmd.Use)
	require.Equal(t, jdNodeSaveAllLong, cmd.Long)
	require.Equal(t, jdNodeSaveAllExample, cmd.Example)
	require.Equal(t, "Recreate the nodes.json", cmd.Short)

	dr := cmd.Flags().Lookup("dry-run")
	require.NotNil(t, dr)
	require.Equal(t, "d", dr.Shorthand)
}
