package v6

import (
	"code.cloudfoundry.org/cli/actor/actionerror"
	"code.cloudfoundry.org/cli/actor/sharedaction"
	"code.cloudfoundry.org/cli/actor/v2action"
	"code.cloudfoundry.org/cli/api/cloudcontroller/ccv2/constant"
	"code.cloudfoundry.org/cli/command"
	"code.cloudfoundry.org/cli/command/flag"
	"code.cloudfoundry.org/cli/command/translatableerror"
	"code.cloudfoundry.org/cli/command/v6/shared"
)

//go:generate counterfeiter . UnbindSecurityGroupActor

type UnbindSecurityGroupActor interface {
	UnbindSecurityGroupByNameAndSpace(securityGroupName string, spaceGUID string, lifecycle constant.SecurityGroupLifecycle) (v2action.Warnings, error)
	UnbindSecurityGroupByNameOrganizationNameAndSpaceName(securityGroupName string, orgName string, spaceName string, lifecycle constant.SecurityGroupLifecycle) (v2action.Warnings, error)
}

type UnbindSecurityGroupCommand struct {
	RequiredArgs    flag.UnbindSecurityGroupArgs `positional-args:"yes"`
	Lifecycle       flag.SecurityGroupLifecycle  `long:"lifecycle" choice:"running" choice:"staging" default:"running" description:"Lifecycle phase the group applies to"`
	usage           interface{}                  `usage:"CF_NAME unbind-security-group SECURITY_GROUP ORG SPACE [--lifecycle (running | staging)]\n\nTIP: Changes require an app restart (for running) or restage (for staging) to apply to existing applications."`
	relatedCommands interface{}                  `related_commands:"apps, restart, security-groups"`

	UI          command.UI
	Config      command.Config
	Actor       UnbindSecurityGroupActor
	SharedActor command.SharedActor
}

func (cmd *UnbindSecurityGroupCommand) Setup(config command.Config, ui command.UI) error {
	cmd.UI = ui
	cmd.Config = config
	cmd.SharedActor = sharedaction.NewActor(config)

	ccClient, uaaClient, err := shared.GetNewClientsAndConnectToCF(config, ui)
	if err != nil {
		return err
	}
	cmd.Actor = v2action.NewActor(ccClient, uaaClient, config)

	return nil
}

func (cmd UnbindSecurityGroupCommand) Execute(args []string) error {
	var err error

	user, err := cmd.Config.CurrentUser()
	if err != nil {
		return err
	}

	var warnings v2action.Warnings

	switch {
	case cmd.RequiredArgs.OrganizationName == "" && cmd.RequiredArgs.SpaceName == "":
		err = cmd.SharedActor.CheckTarget(true, true)
		if err != nil {
			return err
		}

		space := cmd.Config.TargetedSpace()
		cmd.UI.DisplayTextWithFlavor("Unbinding security group {{.SecurityGroupName}} from org {{.OrgName}} / space {{.SpaceName}} as {{.Username}}...", map[string]interface{}{
			"SecurityGroupName": cmd.RequiredArgs.SecurityGroupName,
			"OrgName":           cmd.Config.TargetedOrganization().Name,
			"SpaceName":         space.Name,
			"Username":          user.Name,
		})
		warnings, err = cmd.Actor.UnbindSecurityGroupByNameAndSpace(cmd.RequiredArgs.SecurityGroupName, space.GUID, constant.SecurityGroupLifecycle(cmd.Lifecycle))

	case cmd.RequiredArgs.OrganizationName != "" && cmd.RequiredArgs.SpaceName != "":
		err = cmd.SharedActor.CheckTarget(false, false)
		if err != nil {
			return err
		}

		cmd.UI.DisplayTextWithFlavor("Unbinding security group {{.SecurityGroupName}} from org {{.OrgName}} / space {{.SpaceName}} as {{.Username}}...", map[string]interface{}{
			"SecurityGroupName": cmd.RequiredArgs.SecurityGroupName,
			"OrgName":           cmd.RequiredArgs.OrganizationName,
			"SpaceName":         cmd.RequiredArgs.SpaceName,
			"Username":          user.Name,
		})
		warnings, err = cmd.Actor.UnbindSecurityGroupByNameOrganizationNameAndSpaceName(cmd.RequiredArgs.SecurityGroupName, cmd.RequiredArgs.OrganizationName, cmd.RequiredArgs.SpaceName, constant.SecurityGroupLifecycle(cmd.Lifecycle))

	default:
		return translatableerror.ThreeRequiredArgumentsError{
			ArgumentName1: "SECURITY_GROUP",
			ArgumentName2: "ORG",
			ArgumentName3: "SPACE"}
	}

	cmd.UI.DisplayWarnings(warnings)
	if err != nil {
		switch e := err.(type) {
		case actionerror.SecurityGroupNotBoundError:
			cmd.UI.DisplayWarning("Security group {{.Name}} not bound to this space for lifecycle phase '{{.Lifecycle}}'.",
				map[string]interface{}{
					"Name":      e.Name,
					"Lifecycle": e.Lifecycle,
				})
			cmd.UI.DisplayOK()
			return nil
		default:
			return err
		}
	}

	cmd.UI.DisplayOK()
	cmd.UI.DisplayText("TIP: Changes require an app restart (for running) or restage (for staging) to apply to existing applications.")

	return nil
}
