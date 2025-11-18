# CLA Management Guide

This document explains how the Contributor License Agreement (CLA) system works for Transire and what parts are automated vs. manual.

## CLA Automation with GitHub Actions

### What's Automated by the GitHub Action (contributor-assistant/github-action)

The CLA workflow (`.github/workflows/cla.yml`) automatically handles:

1. **CLA Status Checking**
   - Automatically checks if PR contributors have signed the CLA
   - Posts comments on PRs when CLA signature is required
   - Updates PR status checks based on CLA compliance

2. **Signature Processing**
   - Recognizes when users comment "I have read the CLA Document and I hereby sign the CLA"
   - Automatically adds GitHub usernames to the `cla-signatures/cla.json` file
   - Commits the updated signatures file to the repository
   - Updates PR status to "CLA signed" automatically

3. **Bot Management**
   - Automatically allowlists known bots (dependabot, renovate, etc.)
   - Handles multiple contributors on a single PR

4. **PR Status Updates**
   - Sets GitHub status checks to pass/fail based on CLA compliance
   - Updates comments when all contributors have signed

### What Requires Manual Management

#### 1. Initial Setup (One-time, Already Done)

- ✅ Created CLA workflow file (`.github/workflows/cla.yml`)
- ✅ Created CLA text document (`.github/CLA.md`)
- ✅ Created issue templates for individual/corporate CLAs
- ✅ Created initial `cla-signatures/cla.json` file
- ✅ Updated CONTRIBUTING.md with CLA instructions

#### 2. Corporate CLA Management (Ongoing Manual Tasks)

**Corporate CLAs require more manual oversight:**

- **Review corporate CLA submissions** in GitHub issues
- **Verify authorized signatory** has legal authority to bind the corporation
- **Manually add corporate employees** to the CLA signatures file
- **Monitor employee changes** when corporations update their authorized contributor lists
- **Handle complex corporate structures** (subsidiaries, parent companies)

**Example corporate employee management:**

```json
{
   "signedContributors": [
      {
         "name": "john-doe",
         "id": "12345678",
         "comment_id": "987654321",
         "created_at": "2024-01-15T10:30:00Z",
         "repoId": "123456789",
         "pullRequestNo": "42"
      },
      {
         "name": "jane-smith",
         "id": "87654321",
         "comment_id": "123456789",
         "created_at": "2024-02-01T14:20:00Z",
         "repoId": "123456789",
         "pullRequestNo": "0",
         "corporate": {
            "company": "ACME Corp",
            "authorized_by": "corporate-cla-issue-123"
         }
      }
   ]
}
```

#### 3. Issue Management (Periodic Manual Tasks)

- **Close CLA issues** after processing signatures
- **Respond to questions** about the CLA process
- **Handle edge cases** (name changes, transferred accounts, etc.)
- **Monitor CLA issue templates** and update if needed

#### 4. Repository Management

- **Backup CLA signatures** periodically
- **Monitor the cla-signatures directory** for corruption
- **Update bot allowlists** if new automation tools are added
- **Review and update CLA text** if legal requirements change

## Current Configuration

**Transire uses local CLA signature storage** for simplicity:

- **Signatures stored in**: `cla-signatures/cla.json` (in this repository)
- **No external dependencies**: No separate repository or personal access tokens needed
- **Automatic backup**: CLA signatures are backed up as part of normal repository backups
- **Version controlled**: All signature changes are tracked in git history

### Optional Enhancements

Consider these additional automation options:

#### 1. Separate CLA Signatures Repository

For organizations with multiple repositories, you can:
- Create a dedicated `cla-signatures` repository
- Update the workflow to use `remote-organization-name` and `remote-repository-name`
- Add `PERSONAL_ACCESS_TOKEN` secret with repository write permissions
- Centralize all CLA signatures across projects

#### 2. Integration with External CLA Services

Alternative to the GitHub Action approach:
- **CLA Assistant** (cla-assistant.io) - Web-based CLA signing
- **EasyCLA** (lfx.linuxfoundation.org) - Linux Foundation's CLA management
- **CLAHub** (clahub.com) - Simple web-based CLA signing

#### 3. Advanced Automation

- **Slack/Discord notifications** when CLAs are signed
- **Database integration** for enterprise CLA management
- **LDAP/SSO integration** for corporate contributor verification

## Best Practices for CLA Management

### 1. Regular Maintenance

- **Weekly**: Review and close processed CLA issues
- **Monthly**: Backup CLA signatures file
- **Quarterly**: Review corporate contributor lists with companies

### 2. Security Considerations

- **Verify identity** for corporate CLAs (email domain matching, LinkedIn verification)
- **Monitor for fraudulent** signatures or impersonation
- **Keep signatures secure** - the CLA signatures file contains legally binding agreements

### 3. Documentation

- **Keep clear records** of corporate authorizations
- **Document any manual interventions** in CLA issues
- **Maintain audit trail** of CLA-related decisions

## Troubleshooting Common Issues

### CLA Bot Not Working

1. Check the `PERSONAL_ACCESS_TOKEN` secret is set correctly
2. Verify the bot has write permissions to the repository
3. Check GitHub Action logs for errors

### Missing Signatures

1. Check if the contributor commented the exact phrase
2. Verify the CLA workflow triggered properly
3. Look for typos in the comment

### Corporate Contributor Issues

1. Verify the authorized signatory has submitted a corporate CLA
2. Check that the employee is listed in the corporate CLA issue
3. Manually add corporate employees if the automation missed them

## Summary

**Highly Automated:** Individual CLA signatures, PR status checks, signature storage
**Manual Management Required:** Corporate CLAs, issue triage, edge cases, auditing

The current setup provides excellent automation for individual contributors while requiring reasonable manual oversight for corporate contributors and edge cases.