# RBAC Permission

## Roles

Admin:

- Manage the full system.
- Create, edit, delete users and roles.
- Assign users to teams.
- Configure channels and routing rules.
- View all conversations.

Manager:

- View conversations assigned to staff in teams they manage.
- Reassign chats inside managed teams.
- View team reports.
- Cannot view other teams unless explicitly granted.

Staff:

- View conversations assigned to them.
- Reply only in conversations they can access.
- View allowed customer history.
- Upload and view media in permitted conversations.

Supervisor / Auditor:

- View-only access.
- May view audit logs if granted.
- Must not send replies unless role includes explicit send permission.

## Required Checks

Every API or database query that returns conversation data must answer:

- Can the user view this conversation?
- Can the user send a message in this conversation?
- Can the user view or download this attachment?
- If manager access is used, does the manager actually manage the assigned staff or team?
- Does admin override apply?
- Which audit log should be written?

## Conversation Visibility Rule

Allow view when any condition is true:

- User is admin with `conversation:view_all`.
- User is assigned to the conversation.
- User is a conversation member.
- User is manager of the assigned team or assigned staff and has `conversation:view_team`.
- User is supervisor/auditor with view-only permission for the team/channel.

Deny by default.

## Send Message Rule

Allow send when:

- User can view the conversation.
- Conversation is open or reopened.
- User has `message:send_assigned`, `message:send_team`, or admin override.
- Channel account is enabled and healthy enough for outbound send.

Never allow auditors to send unless explicitly configured.

## Attachment Rule

Allow view/download when:

- User can view the parent conversation.
- Attachment belongs to a message in that conversation.
- Attachment is not expired, quarantined, or blocked.
- Private CDN URLs are signed or proxied through an authorized API.

## Audit Actions

Record at least:

- `conversation.view`
- `message.send`
- `conversation.assign`
- `conversation.transfer`
- `conversation.close`
- `attachment.upload`
- `attachment.download`
- `channel_account.create`
- `channel_account.update`
- `role.update`
- `user.role_assign`

Audit metadata should include actor, resource, previous value, new value, IP, user agent, and request trace ID when available.
