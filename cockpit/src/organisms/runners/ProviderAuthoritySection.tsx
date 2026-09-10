import { PageSection, Stack, Title } from "@patternfly/react-core";
export function ProviderAuthoritySection() {
  return (
    <PageSection aria-labelledby="provider-title">
      <Stack hasGutter>
        <Title headingLevel="h2" id="provider-title">
          Provider authority
        </Title>
        <p>
          Forgejo owns runner registration, tokens, labels, workflows, scheduling,
          results, and history. Soda owns only this machine's account, service, status, capacity,
          and local lifecycle.
        </p>
      </Stack>
    </PageSection>
  );
}
