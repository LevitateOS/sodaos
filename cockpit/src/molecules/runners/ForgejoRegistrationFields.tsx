import { HelperText, HelperTextItem } from "@patternfly/react-core";
import { FormGroup, Stack, TextInput } from "@patternfly/react-core";
import { ExternalLink } from "../../atoms/ExternalLink";
import { forgejoBrowserURL } from "../../runners/ui";
export function ForgejoRegistrationFields({
  busy,
  forgejoURL,
}: {
  busy: boolean;
  forgejoURL: string;
}) {
  return (
    <Stack hasGutter>
      <HelperText>
        <HelperTextItem>
          Create a system runner in{" "}
          <ExternalLink href={`${forgejoBrowserURL(forgejoURL)}/admin/actions/runners`}>
            Forgejo runner administration
          </ExternalLink>
          , then copy its UUID and confidential token here.
        </HelperTextItem>
      </HelperText>
      <FormGroup label="Forgejo runner UUID" fieldId="registration-id" isRequired>
        <TextInput
          id="registration-id"
          name="registration_id"
          isRequired
          isDisabled={busy}
          pattern="[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}"
          autoComplete="off"
        />
      </FormGroup>
      <FormGroup label="Native Forgejo host labels" fieldId="forgejo-labels" isRequired>
        <TextInput
          id="forgejo-labels"
          name="forgejo_labels"
          defaultValue="soda-linux:host"
          isRequired
          isDisabled={busy}
          pattern="[A-Za-z0-9][A-Za-z0-9._-]{0,63}:host(,[A-Za-z0-9][A-Za-z0-9._-]{0,63}:host)*"
          autoComplete="off"
        />
        <HelperText>
          <HelperTextItem>
            Use comma-separated <code>name:host</code> labels. Host jobs run directly as this
            runner's isolated Linux account.
          </HelperTextItem>
        </HelperText>
      </FormGroup>
    </Stack>
  );
}
