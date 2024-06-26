package chainregistry

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/DefiantLabs/cosmos-tax-cli/config"
	"github.com/go-git/go-git/v5"
)

const (
	ChainRegistryGitRepo = "https://github.com/cosmos/chain-registry.git"
)

func UpdateChainRegistryOnDisk(chainRegistryLocation string) error {
	_, err := os.Stat(chainRegistryLocation)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	if os.IsNotExist(err) {
		err = os.Mkdir(chainRegistryLocation, 0o777)
		if err != nil {
			return err
		}
	}

	// git clone repo
	_, err = git.PlainClone(chainRegistryLocation, false, &git.CloneOptions{
		URL: ChainRegistryGitRepo,
	})

	// Check if already cloned
	if err != nil && !errors.Is(err, git.ErrRepositoryAlreadyExists) {
		return err
	} else if errors.Is(err, git.ErrRepositoryAlreadyExists) {
		// Pull if already cloned
		r, err := git.PlainOpen(chainRegistryLocation)
		if err != nil {
			return err
		}

		w, err := r.Worktree()
		if err != nil {
			return err
		}

		err = w.Pull(&git.PullOptions{})
		// Ignore up-to-date error
		if err != nil && !errors.Is(err, git.NoErrAlreadyUpToDate) {
			return err
		}
	}

	return nil
}

func GetAssetMapOnDisk(chainRegistryLocation string, chainRegBlacklist map[string]bool) (map[string]Asset, error) {
	chainRegEntries, err := os.ReadDir(chainRegistryLocation)
	if err != nil {
		return nil, err
	}
	assetMap := make(map[string]Asset)
	for _, entry := range chainRegEntries {
		if entry.IsDir() {
			inBlacklist := chainRegBlacklist[entry.Name()]
			if !inBlacklist {
				path := fmt.Sprintf("%s/%s/assetlist.json", chainRegistryLocation, entry.Name())

				// check if file exists
				_, err := os.Stat(path)
				if err != nil && os.IsNotExist(err) {
					config.Log.Warnf("Chain registry asset list for %s does not exist. Skipping...", entry.Name())
					continue
				} else if err != nil {
					return nil, err
				}

				// load asset list
				jsonFile, err := os.Open(path)
				if err != nil {
					return nil, err
				}

				currAssets := &AssetList{}
				err = json.NewDecoder(jsonFile).Decode(currAssets)
				if err != nil {
					return nil, err
				}

				for _, asset := range currAssets.Assets {
					asset.ChainName = currAssets.ChainName
					if prevEntry, ok := assetMap[asset.Base]; ok {
						config.Log.Warnf("Duplicate asset found for %s in %s. Overwriting entry for %s", asset.Base, currAssets.ChainName, prevEntry.ChainName)
					}
					assetMap[asset.Base] = asset
				}
			}
		}
	}

	return assetMap, nil
}
