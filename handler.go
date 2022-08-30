package main

import (
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/blang/semver"
	"github.com/leodido/go-conventionalcommits"
)

type handler struct {
	options   Options
	ccMachine conventionalcommits.Machine
	git       *gitCli
}

func (h handler) getBranchPrefix() string {
	if h.options.ReleasePrefix != nil {
		return *h.options.ReleasePrefix
	}
	return ""
}

func (h *handler) getReleaseRefs() ([]string, error) {
	if !h.options.SkipFetch {
		if err := h.git.fetch(); err != nil {
			sLogger.Error("failed to run fetch on git repository")
			sLogger.Fatal(err)
		}
	}

	refType := "heads"
	if h.options.Tags {
		refType = "tags"
	}

	refs, err := h.git.listRemoteRefs(refType)
	if err != nil {
		return nil, err
	}

	var releasedRefs []string
	for _, ref := range refs {
		prefixRegex := regexp.MustCompile(fmt.Sprintf("%s/", h.getBranchPrefix()))
		if prefixRegex.MatchString(ref) {
			refSplit := prefixRegex.Split(ref, 2)
			sLogger.Debugf("ref %s matched filter and trim, capturing %s", ref, refSplit[1])
			releasedRefs = append(releasedRefs, refSplit[1])
		}
	}

	return releasedRefs, nil
}

func refsToOrderedScopedVersions(refs []string) map[string][]version {
	allVers := map[string][]version{}
	for _, ref := range refs {
		strVer := ref
		scope := ""
		if len(strings.Split(ref, "/")) > 1 {
			split := strings.Split(ref, "/")
			strVer = split[len(split)-1]
			scope = strings.Join(split[:len(split)-1], "/")
		}

		sver, err := semver.ParseTolerant(strVer)
		if err != nil {
			sLogger.Infof("could not parse %s to semantic version", sver)
			sLogger.Info(err)
			continue
		}

		if _, ok := allVers[scope]; !ok {
			allVers[scope] = []version{}
		}
		allVers[scope] = append(allVers[scope], version{
			ver:  strVer,
			sver: sver,
		})
	}

	for _, vers := range allVers {
		sort.Slice(vers, func(i, j int) bool {
			return vers[i].sver.GT(vers[j].sver)
		})
	}

	return allVers
}

func (h handler) getLastCommitOnRef(ref string) (*string, error) {
	fullRef := h.getBranchPrefix() + "/" + ref
	if !h.options.Tags {
		fullRef = h.git.Remote + "/" + fullRef
	}
	return h.git.getLastCommitOnRef(fullRef)
}

func (h handler) listCommits(ref string) ([]string, error) {
	branch := ""

	if h.options.Branch != nil {
		branch = *h.options.Branch
	} else {
		currentBranch, err := h.git.getCurrentBranch()
		if err != nil {
			sLogger.Warn("failed to get the current git branch")
			return nil, err
		}
		if currentBranch == nil {
			sLogger.Warn("failed to get the current git branch")
			return nil, errors.New("was not able to get the current branch on git")
		}

		branch = *currentBranch
	}

	return h.git.listCommits(ref + ".." + h.git.Remote + "/" + branch)
}

func (h handler) parseConventionalCommit(commitMesage string) (
	conventionalcommits.Message,
	*conventionalcommits.ConventionalCommit,
	error) {

	msg, err := h.ccMachine.Parse([]byte(commitMesage))
	if err != nil {
		return nil, nil, err
	}

	if !msg.Ok() {
		return nil, nil, errors.New("commit did not match conventional commit specification")
	}

	if cc, ok := msg.(*conventionalcommits.ConventionalCommit); ok {
		return msg, cc, nil
	}

	return msg, nil, &FatalError{
		message: "failed to cast conventional commit message to conventional commit type",
	}
}

func (h handler) determineIncrementFromCommits(commits []string) int {
	increment := 0
	for _, commit := range commits {
		message, err := h.git.getCommitMessageBody(commit)
		if err != nil {
			sLogger.Error(err)
			continue
		}
		if message == nil {
			sLogger.Errorf("did not find a commit message body for %s", commit)
			continue
		}

		sLogger.Debugf("try to determine increment from %s", commit)
		sLogger.Debug(*message)

		msg, cc, err := h.parseConventionalCommit(*message)
		if err != nil {
			sLogger.Info(err)
			continue
		}

		if msg.IsBreakingChange() {
			increment = majorIncrement
			break
		}

		switch cc.Type {
		case "feat", "refactor":
			if increment < minorIncrement {
				increment = minorIncrement
			}
		case "fix", "chore", "perf", "docs", "style":
			if increment < patchIncrement {
				increment = patchIncrement
			}
		case "build", "ci", "test":
			if increment < buildIncrement {
				increment = buildIncrement
			}
		default:
			sLogger.Infof("conventional commit type '%s' not implemented", cc.Type)
		}
	}

	return increment
}
