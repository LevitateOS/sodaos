// Concrete protected-collector App/cold transport. Browser mutations still pass
// exact normal-owner command permits; private App authentication stays in memory.
import assert from 'node:assert/strict';
import type {Page, Route} from 'playwright';
import {applicationResponse} from '../../../../frontend/spaces/sodaspaces-application-response';
import {adoptApplication} from '../application';
import {openApplicationOwner} from '../application-owner';
import type {ResolvedQualificationInput} from '../scenario';
import type {ResourceScope} from '../closure';
import {check, rawRun} from './support';
import {readDocker, INNER_READER} from './docker-native';
import {nativeRequest, type DockerBinding, type DockerIO, type Witness} from './docker-binding';
import type {DockerPhaseDetails, DockerPhaseRuntime} from './docker-phases';
import {dockerColdExecutor, preflightDockerCold} from './docker-cold';
import type {ColdDestination, FinalRecoveryRuntime} from './final-state-recovery';
import type {PersistencePermits} from './docker-persistence';
import {
  containerNativeScript,
  requestContainerBytes,
  requestContainerHttp,
  type ContainerHttpTarget,
} from './namespace-http';
import {nativeSpacesMount} from '../../sodaspaces-journey-evidence';

export async function prepareDockerRecovery(
  input: ResolvedQualificationInput,
  details: DockerPhaseDetails,
  io: DockerIO,
  page: Page,
  apiBase: string,
  b: DockerBinding,
  permits: PersistencePermits,
  fixtureBytes: Uint8Array,
  credentials: () => Promise<{email: string; password: string}>
): Promise<Awaited<ReturnType<NonNullable<DockerPhaseRuntime['prepareFinalRecovery']>>>> {
  check(details.cold && input.scenario.applications?.length === 1, 'selected_app_cold_inputs_required');
  await preflightDockerCold(io, details.cold);
  const s = input.scenario,
    selected = s.applications![0]!,
    scope: ResourceScope = {scenario: s, observation: input.observation};
  const current = await readDocker(io, b, 'idle', undefined, scope);
  await io.save('app/' + selected.application_id + '/native-adoption.json', {
    host_uuid: current.host_uuid,
    outer_id: current.outer_id,
    source: current.inner.source,
    payload_sha256: current.inner.payload_sha256,
    application_runtime: current.inner.application_runtime,
    resource_inventory: current.inner.resource_inventory,
  });
  const normal = async (route: string, body?: unknown) =>
    page.evaluate(
      async ({route, body}) => {
        const response = await (document.querySelector('soda-spaces') as any).binding.transport.request(route, {
          method: body === undefined ? 'GET' : 'POST',
          ...(body === undefined ? {} : {headers: {'Content-Type': 'application/json'}, body: JSON.stringify(body)}),
          signal: AbortSignal.timeout(240000),
        });
        if (response.status !== 200) throw Error('application_normal_owner_request_unconfirmed');
        return response.json();
      },
      {route, body}
    );
  const application = await adoptApplication(
    s,
    {
      format: 1,
      checked_at: new Date().toISOString(),
      ready: false,
      checks: ['target_trust', 'native_observation', 'installed_identity', 'installed_images', 'docker_identity'].map(
        (name) => ({name, state: 'pass'})
      ),
      observation: input.observation,
      closure: null,
      native: {
        uuid: current.host_uuid,
        distribution: 'docker',
        source_revision: current.inner.source,
        payload_sha256: current.inner.payload_sha256,
        application_runtime: current.inner.application_runtime,
        docker: {container_id: current.outer_id, container_name: 'sodaos-soda-1', reference: b.image},
      },
      scope: 'read-only readiness; no execution or qualification',
    },
    selected.application_id,
    {get: (route) => normal(route.slice(5)), save: io.save, fixtureBytes}
  );
  check(
    current.inner.application_runtime.some((a: any) => a.id === selected.application_id && a.running),
    'existing_running_application_required'
  );
  const privateOwner = await credentials();
  check(
    typeof privateOwner.email === 'string' &&
      typeof privateOwner.password === 'string' &&
      privateOwner.email.length <= 320 &&
      privateOwner.password.length <= 16384,
    'protected_app_owner_credentials_invalid'
  );
  const address = new URL(s.target.origin).hostname;
  check(/^\d{1,3}(?:\.\d{1,3}){3}$/.test(address), 'private_ipv4_recovery_target_required');
  const target = (destination: 'primary' | ColdDestination, outer: string): ContainerHttpTarget =>
    destination === 'primary'
      ? {container: outer, image: b.image}
      : {container: destination.container, image: b.image, restore_directory: destination.directory};
  const owner = (destination: 'primary' | ColdDestination, outer: string) =>
    openApplicationOwner({
      origin: application.application.origin,
      owner_team: application.owner.team_id,
      credentials: privateOwner,
      transport: (request) =>
        requestContainerHttp(io, target(destination, outer), application.application.origin, address, request),
    });
  const probeOwner = await owner('primary', current.outer_id);
  const record = await probeOwner.ensureRecord(selected.record, {
    save: async (n, v) => {
      await io.save('app/' + selected.application_id + '/' + n, v);
    },
  });
  await io.save('app/' + selected.application_id + '/record.json', record);
  return {
    application,
    probe: {owner: probeOwner, witness: record},
    operations(resources: ResourceScope, witness: Witness): FinalRecoveryRuntime {
      const cold = dockerColdExecutor(b, witness, resources, io, details.cold!);
      let isolated: ColdDestination | undefined, tunnel: ((route: Route) => Promise<void>) | undefined;
      const network: {method: string; path: string; blocked: boolean; status?: number}[] = [];
      const observe = async (destination: ColdDestination, mode: 'idle' | 'snapshot') =>
        JSON.parse(
          await io.sshRead(
            containerNativeScript(
              target(destination, ''),
              'const REQUEST=' +
                JSON.stringify({
                  ...nativeRequest(b, witness),
                  mode,
                  resources: {
                    projects: resources.observation.inventory.projects,
                    applications: resources.observation.inventory.applications,
                  },
                }) +
                ';\n' +
                INNER_READER
            ),
            210000
          )
        );
      const lifecycle = async (
        destination: 'primary' | ColdDestination,
        handoff: typeof application,
        action: 'start' | 'stop'
      ) => {
        const route = 'applications/' + handoff.application.id,
          view = applicationResponse(await normal(route), handoff.application);
        check(
          !view.restore && view.observation?.container === handoff.container_id,
          'current_application_generation_required'
        );
        const wanted = action === 'start';
        if (view.observation.running !== wanted) {
          const body = {
            action,
            revision: handoff.application.revision,
            expected_container: handoff.container_id,
            confirm_stop: !wanted,
          };
          permits.arm(
            'verify-' +
              (destination === 'primary' ? 'primary' : 'isolated') +
              '-' +
              action +
              '-application-' +
              handoff.application.id,
            apiBase + '/' + route + '/actions',
            body
          );
          try {
            await normal(route + '/actions', body);
          } catch {
            /* Read this exact effect; never repeat the POST. */
          }
        }
        const native =
          destination === 'primary'
            ? (await readDocker(io, b, 'idle', witness, resources)).inner
            : await observe(destination, 'idle');
        check(
          native.application_runtime.some(
            (a: any) =>
              a.id === handoff.application.id &&
              a.container_id === handoff.container_id &&
              a.running === wanted &&
              (wanted || (a.unit_state === 'inactive' && a.main_pid === 0 && a.control_pid === 0))
          ),
          'native_application_lifecycle_unconfirmed'
        );
      };
      return {
        ...cold,
        async openApplication(destination, handoff) {
          await lifecycle(destination, handoff, 'start');
          return owner(
            destination,
            handoff.transport.distribution === 'docker' ? handoff.transport.outer_container_id : ''
          );
        },
        stopApplication: (destination, handoff) => lifecycle(destination, handoff, 'stop'),
        async startAndObserve(destination) {
          check(
            destination.network === 'none' &&
              destination.state === 'created' &&
              destination.directory === details.cold!.restore_directory,
            'selected_isolated_restore_required'
          );
          isolated = destination;
          const program =
            'const TARGET=' +
            JSON.stringify(destination) +
            ',ADDRESS=' +
            JSON.stringify(address) +
            ';\n' +
            String.raw`
const fs=require('node:fs'),cp=require('node:child_process');const need=x=>{if(!x)throw Error()};const d=a=>cp.execFileSync('/usr/bin/docker',['--host','unix:///var/run/docker.sock',...a],{encoding:'utf8',timeout:300000,maxBuffer:65536,stdio:['ignore','pipe','pipe']});try{need(process.getuid()===0);const c=JSON.parse(d(['inspect',TARGET.container]))[0];need(c.Id===TARGET.container&&c.Config.Image===TARGET.image&&c.State.Status==='created'&&c.HostConfig.NetworkMode==='none'&&c.HostConfig.PidMode!=='host'&&c.HostConfig.IpcMode!=='host'&&c.Mounts.length===3&&['etc','home','var'].every(n=>c.Mounts.some(m=>m.Type==='bind'&&m.Source===TARGET.directory+'/'+n&&m.Destination==='/'+n)));
const fd=fs.openSync(TARGET.directory+'/boot.request.json',fs.constants.O_WRONLY|fs.constants.O_CREAT|fs.constants.O_EXCL|fs.constants.O_NOFOLLOW,384);try{fs.writeFileSync(fd,JSON.stringify({container:TARGET.container,network:'none',automatic_retry:false}));fs.fsyncSync(fd)}finally{fs.closeSync(fd)}d(['start',TARGET.container]);
d(['exec',TARGET.container,'/bin/sh','-eu','-c','ip address replace "$1/32" dev lo','recovery-address',ADDRESS]);
let manager=false;for(let i=0;i<60;i++){const r=cp.spawnSync('/usr/bin/docker',['--host','unix:///var/run/docker.sock','exec',TARGET.container,'systemctl','show','--property=SystemState','--value'],{stdio:'ignore',timeout:10000});if(r.status===0){manager=true;break}Atomics.wait(new Int32Array(new SharedArrayBuffer(4)),0,0,1000)}need(manager);d(['exec',TARGET.container,'systemctl','restart','soda-proxy.service']);
let ready=false;for(let i=0;i<60;i++){const r=cp.spawnSync('/usr/bin/docker',['--host','unix:///var/run/docker.sock','exec',TARGET.container,'systemctl','is-active','--quiet','soda-host.service','soda-dashboard.service','soda-identity.service','soda-proxy.service','soda-postgres.service','forgejo.service'],{stdio:'ignore',timeout:10000});if(r.status===0){ready=true;break}Atomics.wait(new Int32Array(new SharedArrayBuffer(4)),0,0,2000)}need(ready);console.log(JSON.stringify({container:TARGET.container,network:'none',services_ready:true}));}catch{console.error('isolated_boot_unconfirmed_no_retry');process.exitCode=1;}`;
          await io.save('supplemental/recovery/boot.request.json', {
            container: destination.container,
            automatic_retry: false,
          });
          await io.sshRead(program, 300000);
          tunnel = async (route) => {
            const request = route.request(),
              url = new URL(request.url());
            check(network.length < 512, 'bounded_isolated_browser_requests');
            const row = {method: request.method(), path: url.pathname, blocked: false} as (typeof network)[number];
            network.push(row);
            if (url.origin !== s.target.origin || rawRun(url.href) || !['GET', 'POST'].includes(request.method())) {
              row.blocked = true;
              return route.abort('blockedbyclient');
            }
            if (request.method() === 'POST' && !(await permits.admit(request))) {
              row.blocked = true;
              return route.abort('blockedbyclient');
            }
            const headers = await request.allHeaders();
            for (const name of ['accept-encoding', 'connection', 'content-length']) delete headers[name];
            headers['accept-encoding'] = 'identity';
            const response = await requestContainerBytes(io, target(destination, ''), s.target.origin, address, {
              route: url.pathname + url.search,
              method: request.method() as 'GET' | 'POST',
              body: request.postData(),
              headers,
            });
            row.status = response.status;
            await route.fulfill({
              status: response.status,
              headers: {
                ...response.headers,
                ...(response.set_cookie.length ? {'set-cookie': response.set_cookie.join('\n')} : {}),
              },
              body: Buffer.from(response.body_base64, 'base64'),
            });
          };
          await page.route('**/*', tunnel);
          await page.goto(s.target.origin + '/workspace?repository=' + s.factory[0]!.subject.repository_id);
          await nativeSpacesMount(page);
          return observe(destination, 'snapshot');
        },
        observeIsolated: (destination) => observe(destination, 'snapshot'),
        async stopIsolated(destination) {
          assert(isolated?.container === destination.container);
          try {
            await io.sshRead(
              'const TARGET=' +
                JSON.stringify(destination) +
                ';\n' +
                String.raw`const cp=require('node:child_process');try{if(process.getuid()!==0)throw Error();const d=a=>cp.execFileSync('/usr/bin/docker',['--host','unix:///var/run/docker.sock',...a],{encoding:'utf8',timeout:300000,maxBuffer:16384,stdio:['ignore','pipe','pipe']});const c=JSON.parse(d(['inspect',TARGET.container]))[0];if(c.Id!==TARGET.container||c.Config.Image!==TARGET.image||c.HostConfig.NetworkMode!=='none')throw Error();if(c.State.Running)d(['stop','--time','300',TARGET.container]);const after=JSON.parse(d(['inspect',TARGET.container]))[0];if(after.State.Running||!['exited','created'].includes(after.State.Status)||after.State.OOMKilled||(after.State.Status==='exited'&&![0,130].includes(after.State.ExitCode)))throw Error();console.log(JSON.stringify({container:TARGET.container,stopped:true,storage_preserved:true}));}catch{console.error('isolated_retirement_unconfirmed');process.exitCode=1;}`,
              330000
            );
          } finally {
            if (tunnel) await page.unroute('**/*', tunnel);
            await io.save('supplemental/recovery/browser-network.json', {
              format: 1,
              requests: network,
              bodies_or_credentials_saved: false,
            });
          }
        },
      };
    },
  };
}
