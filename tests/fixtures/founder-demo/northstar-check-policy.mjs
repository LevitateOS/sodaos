// Protected manager and Finance outcomes, history and retained-record assertions.
import assert from 'node:assert/strict';
export function policyChecks(context, actors) {
  const {APP} = context;
  const {read, denied, submit, ref, operation, row} = actors;
  function historySections(html) {
    // Each History disclosure contains nested operation disclosures. Match the
    // balanced outer region, rather than ending at the first operation's close.
    const stack = [],
      sections = [];
    for (const tag of html.matchAll(/<details\b[^>]*>|<\/details>/g)) {
      if (!tag[0].startsWith('</')) stack.push(tag.index);
      else {
        const start = stack.pop();
        assert(start !== undefined, 'Unbalanced application disclosures.');
        const section = html.slice(start, tag.index + tag[0].length);
        if (section.match(/<summary\b[^>]*>([^<]*)<\/summary>/)?.[1] === 'History') sections.push(section);
      }
    }
    assert.equal(stack.length, 0, 'Unclosed application disclosure.');
    return sections;
  }

  async function baselineChecks(people) {
    const {maya, alex, jordan, evan} = people;
    assert.equal((await read(alex)).length, 0, 'Fresh acceptance state must be empty.');
    for (const inputs of [
      {title: 'Invalid cost', reason: 'Rejected check', amount: {minor: '0', currency: 'EUR'}},
      {title: 'Invalid currency', reason: 'Rejected check', amount: {minor: '100', currency: 'USD'}},
      {title: '', reason: 'Rejected check', amount: {minor: '100', currency: 'EUR'}},
      {title: 'Missing reason', reason: '', amount: {minor: '100', currency: 'EUR'}},
    ])
      await denied(alex, 'request_equipment', inputs, alex);
    let monitor = await submit(alex, 'Retained monitor', '42000');
    assert.equal(monitor.data.stage, 'waiting_manager');
    assert.equal(monitor.data.needs_finance, false);
    const pending = await submit(alex, 'Pending pre-change workstation', '150000');
    assert.equal(pending.data.needs_finance, false, 'Original manager-only route must be saved at submission.');
    assert.equal((await read(evan)).length, 0, 'Unrelated employee can read private requests.');
    const evanPage = await evan.send('/', undefined, {accept: 'text/html'});
    assert(
      !evanPage.text.includes('Retained monitor') && !evanPage.text.includes('Pending pre-change workstation'),
      'HTML disclosed unrelated requests.'
    );
    await denied(alex, 'approve', {request: ref(monitor)}, alex, monitor.id);
    await denied(maya, 'deliver', {request: ref(monitor)}, alex, monitor.id);
    const own = await submit(jordan, 'Manager own request', '25000');
    await denied(jordan, 'approve', {request: ref(own)}, jordan, own.id);
    assert.equal((await operation(jordan, 'approve', {request: ref(monitor)})).status, 200);
    monitor = await row(maya, monitor.id);
    assert.equal(monitor.data.stage, 'fulfillment');
    assert.equal((await operation(maya, 'deliver', {request: ref(monitor)})).status, 200);
    monitor = await row(alex, monitor.id);
    assert.equal(monitor.data.stage, 'delivered');
    let declined = await submit(alex, 'Declined duplicate request', '9000');
    assert.equal(
      (await operation(jordan, 'decline', {request: ref(declined), note: 'Existing equipment is sufficient'})).status,
      200
    );
    declined = await row(alex, declined.id);
    assert.equal(declined.data.stage, 'declined');
    assert.equal(declined.data.decision_note, 'Existing equipment is sufficient');
    await denied(maya, 'deliver', {request: ref(declined)}, alex, declined.id);
    const page = await maya.send('/', undefined, {accept: 'text/html'});
    assert.equal(page.status, 200);
    const histories = historySections(page.text);
    assert(
      histories.some((value) =>
        ['request_equipment', 'approve', 'deliver'].every((name) => value.includes(APP + '.' + name))
      ),
      'Actual saved request/approval/delivery history missing.'
    );
    assert(!page.text.includes('data-can-history="unavailable"'), 'History is unavailable.');
    return {monitor, pending};
  }

  async function financeChecks(people, retained) {
    const {maya, alex, jordan, robin, evan} = people;
    assert.deepEqual(
      await row(alex, retained.monitor.id),
      retained.monitor,
      'Delivered v1 record changed after update.'
    );
    assert.deepEqual(
      await row(alex, retained.pending.id),
      retained.pending,
      'Pending v1 record or approval route changed after update.'
    );
    assert.equal((await read(evan)).length, 0, 'Update disclosed private records.');
    let boundary = await submit(alex, 'Exactly EUR1000', '100000');
    let above = await submit(alex, 'Above EUR1000', '100100');
    assert.equal(boundary.data.needs_finance, false);
    assert.equal(above.data.needs_finance, true);
    await denied(jordan, 'approve', {request: ref(above)}, alex, above.id);
    await denied(maya, 'deliver', {request: ref(above)}, alex, above.id);
    assert.equal((await operation(jordan, 'approve', {request: ref(boundary)})).status, 200);
    boundary = await row(maya, boundary.id);
    assert.equal(boundary.data.stage, 'fulfillment');
    assert.equal((await operation(jordan, 'approve_finance_route', {request: ref(above)})).status, 200);
    above = await row(robin, above.id);
    assert.equal(above.data.stage, 'waiting_finance');
    await denied(maya, 'approve_finance', {request: ref(above)}, alex, above.id);
    await denied(maya, 'deliver', {request: ref(above)}, alex, above.id);
    assert.equal((await operation(robin, 'approve_finance', {request: ref(above)})).status, 200);
    above = await row(maya, above.id);
    assert.equal(above.data.stage, 'fulfillment');
    assert.equal((await operation(maya, 'deliver', {request: ref(above)})).status, 200);
    assert.equal((await row(alex, above.id)).data.stage, 'delivered');
    let ownFinance = await submit(robin, 'Finance own request', '100100');
    assert.equal((await operation(jordan, 'approve_finance_route', {request: ref(ownFinance)})).status, 200);
    ownFinance = await row(robin, ownFinance.id);
    await denied(robin, 'approve_finance', {request: ref(ownFinance)}, maya, ownFinance.id);
    let declined = await submit(alex, 'Finance declined request', '100100');
    assert.equal((await operation(jordan, 'approve_finance_route', {request: ref(declined)})).status, 200);
    declined = await row(robin, declined.id);
    assert.equal(
      (await operation(robin, 'decline_finance', {request: ref(declined), note: 'Outside the equipment budget'}))
        .status,
      200
    );
    declined = await row(alex, declined.id);
    assert.equal(declined.data.stage, 'declined');
    assert.equal(declined.data.decision_note, 'Outside the equipment budget');
    await denied(maya, 'deliver', {request: ref(declined)}, alex, declined.id);
    // A pending pre-change EUR1500 request still follows its recorded old route.
    assert.equal((await operation(jordan, 'approve', {request: ref(retained.pending)})).status, 200);
    assert.equal((await row(maya, retained.pending.id)).data.stage, 'fulfillment');
    assert.deepEqual(await row(alex, retained.monitor.id), retained.monitor);
    return {boundary: boundary.id, above: above.id};
  }

  return {baselineChecks, financeChecks};
}
