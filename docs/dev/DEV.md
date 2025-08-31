# Development Notes

### Database Conventions
separate developers should maintain separate databases to avoid conflicting database features

individual developer databases should follow the naming convention: marathondev_name

for example:

```
marathondev_johnsmith
```

### Naming Conventions
branches will be named in the following style

```
MAIN_BRANCH-DEVELOPER_NAME-NOTES-NOTES-ETC
```

for example:

```
dev-johnsmith-options-parsing
```

This is a branch, created by developer "johnsmith" from the "dev" branch. The branch focuses on options parsing.

### Workflow
For each issue, each developer should either build on their own unique dev-name branch, or create a new branch from their
own branch, such as dev-name-fix. 

Create a new pull request from the developer's branch to the dev branch after completing a task. Each pull request should have at least one reviewer. 

Reviewers should review the code thoroughly, do not be "too nice."

Pull requests made on the DEV branch will automatically trigger a run of the unit testing suite. Make sure all tests pass or are accounted for before merging a pull request. 





