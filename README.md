# **changehelper** - example chg

[![Build](https://github.com/marmotherder/changehelper/actions/workflows/go.yml/badge.svg)](https://github.com/marmotherder/changehelper/actions/workflows/go.yml) [![Go Report Card](https://goreportcard.com/badge/github.com/marmotherder/changehelper)](https://goreportcard.com/report/github.com/marmotherder/changehelper)

The changehelper tool is a cli designed to handle most semantic versioning scenarios via a series of easy to use commands. The tool follows the standards as defined in the [semantic versioning](https://semver.org/) specification.

The tool itself is generated as a native platform binary, and can be used to handle the updates to a [CHANGELOG.md](https://keepachangelog.com/en/1.0.0/) file, as well as release branches in mono and poly repos within git.


Support has been added for [conventional commits](https://www.conventionalcommits.org/en/v1.0.0/), with both versions, and updates to a changelog now being supported via automatically parsing conventional commits.

## **Basic usage**

The command line has five core commands outlined here:
* `new-version` - Interactive prompt to generate a new release, which will be loaded into a changelog file
* `print-current-version` - Prints the current version as detected in the changelog file
* `print-unreleased-version` - Prints the unreleased version as specified in the changelog file. If no unreleased change is defined, it will exit with code 1
* `print-current-change` - Prints the current change text as detected in the changelog file
* `print-unreleased-change` - Prints the unreleased change text as specified in the changelog file. If no unreleased change is defined, it will exit with code 1
* `update` - Will update the changelog file with a release version. This is either done from an unreleased version present in the file already, or, it will generate a version based on conventional commits
* `release` - Will release the changes in the changelog (and any other files added externally) to git, pushing the changes to the specified trunk branch, and generating/updating release branches/tags as appropriate for the release
* `update-and-release` - Runs both version and release as a single command one after the other
* `enforce-unreleased` - Validates that there is a pending unreleased change in the changelog, else will exit with code 1
* `enforce-conventional-commits` - Enforce that all commits adhere to conventional commit standards, or will exit with code 1
* `version` - Print the version of the tool

The following are global options that can be set:
```
* -l --log-level        Sets the log level of the application. To increase verbosity, specify the flag multiple times, eg. '-ll' will set the application to 'WARN' level. The following are the logging levels supported by the tool. Defaults to 'FATAL' level
    FATAL   = 0
    ERROR   = 1
    WARN    = 2
    INFO    = 3
    DEBUG   = 4
* -f --changelog-file   The location (relative or absolute) of the desired changelog file to parse. Defaults to './CHANGELOG.md'
* -h --help             Print the help options for the selected operation
```

### **new-version**

new-version command will attempt to update the changelog file with the desired release.

By default, the command will scan through the git branch, and attempt to detect change level by reading conventional commits. By default, it will attempt to run a diff between the HEAD of the current git branch, and the last detected change to the changelog file specified.

Alternatively, if no conventional commits are specified, it will attempt to run interactively, and prompt for the desired incrementation level. It will then, unless overridden, attempt to determine which files have changed, and prompt for text to enter into the changelog as appropriate. 

All options are optional for this command, if none are provided, the tool will run interactively, but attempt to reconcile the version from conventional commits non interactively first.

### **Interactive**

By default, the tool is interactive, however, the parsing and creation of a new version via conventional commits is non interactive. Use the option flag `-n` to make sure all interactions are non interactive, otherwise, the tool will fall back to interactive if it can't resolve commits to its satisfaction.

#### **Options**

```
* -b --git-branch                   The branch to run against. By default, this isn't set, and will use the currently checked out branch locally
* -w --git-workdir                  The working directory for git, by default this is the same directory as the tool is run in
* -s --skip-git-checkout            Should the checkout of a git branch be skipped? If a git branch is explicitly provided, and this is toggled, the resulting git lookup behaviour may not be as expected
* -i --increment                    The incrementation level for the application, only MAJOR, MINOR, and PATCH are supported
* -o --force                        Force new version, even if a pending release is present, defaults to false
* -m --manual                       Disable all automation, so conventional commits and/or changes from git will not be resolved
* -n --non-interactive              Only allow the tool to run without any interactive prompting
* -g --ignore-conventionalcommits   When running in an automation fashion, skip attempting to load/parse conventional commits
* -a --added            List of added in the relase (provide the flag multiple times for every line)
* -c --changed          List of changed in the relase (provide the flag multiple times for every line)
* -d --deprecated       List of deprecated in the relase (provide the flag multiple times for every line)
* -r --removed          List of removed in the relase (provide the flag multiple times for every line)
* -x --fixed            List of fixed in the relase (provide the flag multiple times for every line)
* -s --security         List of security changed in the relase (provide the flag multiple times for every line)
* -d --depth                How deep to check down the git tree when looking for conventional commits. If set, it will override the default behaviour, which is reading all commits after the last change to the changelog file
```

### **print-current-version**

print-current-version has no additional options, and will simply print the current version in the changelog. In the event of an unreleased version being present, it will print the most recent released version.

### **print-unreleased-version**

print-unreleased-version will check the changelog file to validate if an unreleased version is present.

If an unreleased version is not present, it will always exit with code 1.

If an unreleased version is present, it will scan through the changelog file, and print the expected next version when released, based on the rules/setup in the changlog file. For example, if the current version in `./CHANGELOG.md` is `1.0.0`, and there is an unreleased version with the change level as `PATCH`, this will print version `1.0.1`.

The command can, optionally, evaluate versions against git release branches, in which both the changelog file, and the git release branches will inform the next version. For example, the version in `./CHANGELOG.md` is `1.0.0`, there is a branch in git as `release/1.0.1`, and there is an unreleased version with the change level as `PATCH`, then this will print `1.0.2`.

### **print-current-change**

print-current-change has no additional options, and will simply print the current change text in the changelog. In the event of an unreleased version being present, it will print the most recent released version.

### **print-unreleased-change**

print-unreleased-change will check the changelog file to validate if an unreleased version is present. If it is, it will print the text for that unreleased version

### **update**

update will update the changelog file with the unreleased version specified in the file, if present.

If an unreleased version is not present, it will always exit with code 1.

The tool will at fist, attempt to read the changelog file to determine a pending version in that file. If nothing is present, it will attempt to resolve the changes via conventional commits.

#### **Options**

```
* -e --git-evaluate     Should the tool attempt to evaluate against git as part of determining versions, defaults to false
* -w --git-workdir      The location of the git working directory, eg. the location of the '.git' folder, defaults to './'
* -r --git-prefix       The prefix for release branches in git for the tool to lookup, defaults to 'release'
* -t --use-tags         Use tags instead of branches to evaluate the git changes
* -b --git-branch       The branch to run against. By default, this isn't set, and will use the currently checked out branch locally
* -d --depth                How deep to check down the git tree when looking for conventional commits. If set, it will override the default behaviour, which is reading all commits after the last change to the changelog file
```

### **release**

release will release the changes to the changelog file (and any others set with git add) to git trunk branch, and update/create release branches/tags specific to the new release. To this end, this command expects an updated and formatted changelog file at a minimum.

#### **Options**

```
* -e --git-evaluate             Should the tool attempt to evaluate against git as part of determining versions, defaults to false
* -w --git-workdir              The location of the git working directory, eg. the location of the '.git' folder, defaults to './'
* -r --git-prefix               The prefix for release branches in git for the tool to lookup, defaults to 'release'
* -t --use-tags                 Release to tags instead of branches
* -b --git-branch               The trunk branch used to commit changes to as the source of truth, defaults to 'main'
* -s --skip-git-checkout        Should the checkout of a git branch be skipped? If a git branch is explicitly provided, and this is toggled, the resulting git lookup behaviour may not be as expected
* -n --non-interactive          Only allow the tool to run without any interactive prompting
* -m --git-commit-message       Message for the git commit, %s can be used in the message to substitute with the version, defaults to '[skip ci] Release version %s'
* -r, --release-file            Additional files in the repository to add to the relase
* -v --version-prefix           Prefix of the version tag/branches, defaults to 'v'
```

### **update-and-release**

update-and-release runs update, then release commands in sequence. It shares all options with those two commands, and no additional ones

### **enforce-unreleased**

Will scan through the changlog file, and look for a pending release. Will exit with 0 withh no extra information if a pending release is present

### **enforce-conventional-commits**

Will scan through the specified git branch/setup, and check that conventional commits are present. Will error out if no commits can be found.

By default, will attempt to resolve commits after the last change to the changelog file.

#### **Options**

```
* -e --git-evaluate     Should the tool attempt to evaluate against git as part of determining versions, defaults to false

* -r --git-prefix       The prefix for release branches in git for the tool to lookup, defaults to 'release'
* -t --use-tags         Use tags instead of branches to evaluate the git changes




* -b --git-branch           The branch to run against. By default, this isn't set, and will use the currently checked out branch locally
* -w --git-workdir          The location of the git working directory, eg. the location of the '.git' folder, defaults to './'
* -s --skip-git-checkout    Should the checkout of a git branch be skipped? If a git branch is explicitly provided, and this is toggled, the resulting git lookup behaviour may not be as expected
* -d --depth                How deep to check down the git tree when looking for conventional commits. If set, it will override the default behaviour, which is reading all commits after the last change to the changelog file
```

### **version**

Print the curent version of the tool, no options.
