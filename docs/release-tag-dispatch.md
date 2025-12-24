# Release Tag Dispatch Workflow

When creating a new tag using the repository’s default `GITHUB_TOKEN` to perform tasks on behalf of GitHub Actions, events triggered by the `GITHUB_TOKEN` **will not** start a new workflow run.
To work around this, you can set up an **SSH key** to fetch the repository in the pipeline instead of relying on `GITHUB_TOKEN`.

## Steps

1. **Generate a new OpenSSH key**:

   ```sh
   ssh-keygen -t ed25519 -f id_ed25519 -N "" -q -C ""
   ```

   This creates:

   - A private key file: `id_ed25519`
   - A public key file: `id_ed25519.pub`
     in the current working directory.

2. **Add the private key** to the repository’s **Secrets** in GitHub.

   - Name it: `SSH_KEY`

3. **Add the public key** to the repository’s **Deploy Keys** in GitHub.

   - Name it: `SSH_KEY`
   - Enable `Allow write access` if you plan to push changes.

4. Update Repository Settings

   1. Go to your GitHub repository.
   2. Click **Settings** (top tab) -> **Actions** -> **General**.
   3. Scroll down to **Workflow permissions**.
   4. Toggle the checkbox: **"Allow GitHub Actions to create and approve pull requests"**.
   5. Click **Save**.

5. **Update your GitHub Actions workflow** to use the SSH key:
   ```yaml
   jobs:
     tag-changelog:
       runs-on: ubuntu-22.04
       permissions:
         contents: write
         issues: write
         pull-requests: write
       steps:
         - uses: actions/checkout@v4
           with:
             ssh-key: ${{ secrets.SSH_KEY }}
             fetch-depth: 0
   ```
