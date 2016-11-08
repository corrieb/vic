# Virtual Container Host Debug Options #

The command line utility for vSphere Integrated Containers Engine, `vic-machine`, provides a `debug` command that allows you to enable SSH access to the virtual container host endpoint VM, set a password for the root user account, and upload a key file for automatic public key authentication. 

If you authorize SSH access to the virtual container host endpoint VM, you can edit system configuration files that you cannot edit by running `vic-machine` commands.

**NOTE**: Modifications that you make to the configuration of the virtual container host endpoint VM do not persist if you reboot the VM.

The `vic-machine debug` command includes the following options in addition to the common options described in [Common `vic-machine` Options](common_vic_options.md).

### `--enable-ssh` ###

Short name: `--ssh`

Enable an SSH server in the virtual container host endpoint VM. The `sshd` service runs until the virtual container host endpoint VM reboots. The `--enable-ssh` takes no arguments.

<pre>--enable-ssh</pre>

### `--rootpw` ###

Short name: `--pw`

Set a new password for the root user account on the virtual container host endpoint VM.

**IMPORTANT**: If you set a password for the virtual container host endpoint VM, this password does not persist if you reboot the VM. You must run vic-machine debug to reset the password each time you reboot the virtual container host endpoint VM.

Wrap the password in single quotes (Linux or Mac OS) or double quotes (Windows) if it includes special characters.

<pre>--rootpw '<i>new_p@ssword</i>'</pre>

### `--authorized-key` ###

Short name: `--key`

Upload a public key file to `/root/.ssh/authorized_keys` folder in the endpoint VM to implement public authentication when accessing the virtual container host endpoint VM. Include the name of the `*.pub` file in the path.

<pre>--authorized-key <i>path_to_public_key_file</i>/<i>key_file</i>.pub</pre>