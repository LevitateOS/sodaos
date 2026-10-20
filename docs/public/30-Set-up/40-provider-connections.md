# Connect an AI provider account

Connect your supported account when you want to launch a managed AI CLI under your own authorization. Provider connections are personal, encrypted broker records; they are separate from Forgejo sign-in, Project membership and the operator's Factory sponsorship.

Soda supports Codex ChatGPT and Muse Code subscription device sign-in. Open your provider connection controls, choose the provider and complete its native device sign-in. Confirm the connection appears as connected before launching the corresponding CLI. When more than one authorized connection is available, select the intended connection for the run. If the provider rejects a credential or requires renewed sign-in, reconnect that account through the same controls.

Subscription limits and provider quota remain with the provider account. Soda-managed execution uses subscription authentication; an expired or rejected credential or exhausted subscription must stop and explain the required action. Soda does not silently switch to API billing, another account, another provider or another authentication method. A personal managed CLI may use only a connection and Project grant authorized to its owner.

The broker keeps credentials out of prompts and shared Project files. Do not paste credentials into a prompt, terminal transcript, repository, or shared configuration. The managed CLI receives a restricted execution credential for its authorized run; this does not give teammates your account or a credential they can use for a personal takeover.

If a connection needs attention, reconnect it using the provider's sign-in flow. To remove it, explicitly disconnect or revoke it in provider controls. Disconnect first denies new use and retires its managed executions; if retirement cannot be confirmed, the broker keeps the connection fenced from reuse until the outcome is reconciled. Removing a grant can affect only the executions covered by that grant; it does not erase Project files. A rejected credential is a connection problem, while an ordinary CLI error is not proof that sign-in failed.

Provider account enrollment does not authorize autonomous Factory work or pay its ongoing costs. The repository's Factory policy, appliance capacity and explicit sponsorship are configured separately in [Factory authorization](../40-Factory/20-factory-authorization.md). For human terminal and Project setup, provider connection is optional; see [First connection](50-first-connection.md).
