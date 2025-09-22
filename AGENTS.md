# Sonara Project - Cursor AI Assistant Rules

## Core Reference Files

### @project_overview.md
- **Purpose**: Master reference document containing complete technical specifications, architecture, and implementation details for the entire Sonara project
- **Usage**: Always consult this file for technical details, API specifications, database schemas, and architectural decisions
- **Content**: Comprehensive 1250+ line document with code examples, API endpoints, database schemas, and implementation patterns
- **Authority**: Definitive source for "how the system should work"

### @week_1_tickets.md
- **Purpose**: Current sprint backlog and task definitions for Week 1 (Foundation Sprint)
- **Usage**: Reference this for what work needs to be done now and task priorities
- **Content**: Detailed tickets with estimates, acceptance criteria, and implementation guidance
- **Authority**: Source of truth for current development tasks and sprint goals

### @project_state.md
- **Purpose**: Live tracking of current project progress, completed work, and current status
- **Usage**: Check this first to understand what's been done and what the current state is
- **Content**: Concise status updates, completed tickets, current blockers, and progress indicators
- **Authority**: Real-time project status - update this when work is completed or state changes

## State Management Rules

### When to Update @project_state.md
Update project_state.md when:
- ✅ Completing a ticket or task from @week_1_tickets.md
- 🔄 Starting work on a new major component
- ❌ Encountering blockers, errors, or issues that prevent progress
- 📝 Making significant architectural or implementation decisions
- 🎯 Changing priorities or sprint scope
- ✅ Verifying that implemented features work as expected

### How to Update @project_state.md
- **Don't rewrite the entire file** - only update what's changed
- **Be specific and actionable** - include concrete details about what was done
- **Include timestamps** - use format: "2025-01-XX: [description]"
- **Use status indicators** - ✅ completed, 🔄 in progress, ❌ blocked, ⏳ planned
- **Reference ticket numbers** - link back to @week_1_tickets.md tasks
- **Keep it concise** - focus on current state, not historical details

### State Update Format
```
2025-01-XX: ✅ COMPLETED [TICKET-ID] - Brief description of what was accomplished
2025-01-XX: 🔄 STARTED [TICKET-ID] - Current work in progress
2025-01-XX: ❌ BLOCKED [TICKET-ID] - Issue encountered and current status
```

## Development Workflow

### Before Starting Work
1. **Check @project_state.md** - Understand current progress and any blockers
2. **Review @week_1_tickets.md** - Identify next priority task
3. **Consult @project_overview.md** - Understand technical requirements and patterns
4. **Plan implementation** - Consider how it fits with existing architecture

### During Implementation
1. **Follow established patterns** from @project_overview.md
2. **Test thoroughly** - Aim for the 80% coverage target mentioned in overview
3. **Document decisions** - Update state if architectural choices are made
4. **Handle errors gracefully** - Update state if blockers encountered

### After Completing Work
1. **Update @project_state.md** - Mark task as completed with details
2. **Verify functionality** - Ensure the feature works as specified
3. **Check for follow-up tasks** - See if completion unlocks other work
4. **Clean up** - Remove temporary files, update documentation if needed

## Code Quality Standards

### From @project_overview.md Requirements
- **Backend**: Go 1.21+, Huma v2, Chi router, PostgreSQL, AWS S3, OpenAI integration
- **Frontend**: React 18, TypeScript, Vite, Tailwind CSS, British Racing Green theme
- **Testing**: 80% coverage target, testify, testcontainers-go, Vitest
- **Deployment**: Railway, Docker, automated CI/CD

### Always Verify
- ✅ Code compiles and runs
- ✅ Tests pass (aim for 80% coverage)
- ✅ Follows established patterns from @project_overview.md
- ✅ Updates @project_state.md when work is completed
- ✅ No breaking changes without updating dependent components

## Communication Style

### Be Proactive
- Suggest next logical steps when current task is complete
- Point out potential issues or improvements
- Ask clarifying questions when requirements are ambiguous

### Be Efficient
- Use the reference files to avoid re-explaining established concepts
- Focus on implementation details rather than repeating specifications
- Update project state promptly when work status changes

### Stay Focused
- Work through @week_1_tickets.md systematically
- Don't jump ahead to future weeks' features
- Keep current sprint goals in mind
- Update @project_state.md to track progress toward Week 1 completion
