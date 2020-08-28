'use strict'
const config = require('conventional-changelog-conventionalcommits');

module.exports = config({
  "types" : [
    { type: 'feat', section: 'Features' },
    { type: 'feature', section: 'Features' },
    { type: 'fix', section: 'Bug Fixes' },
    { type: 'perf', section: 'Performance Improvements' },
    { type: 'revert', section: 'Reverts' },
    { type: 'docs', section: 'Documentation'},
    { type: 'style', section: 'Styles'},
    { type: 'chore', section: 'Miscellaneous Chores', hidden: true },
    { type: 'refactor', section: 'Code Refactoring'},
    { type: 'test', section: 'Tests'},
    { type: 'build', section: 'Build System'},
    { type: 'ci', section: 'Continuous Integration'}
  ]
})
