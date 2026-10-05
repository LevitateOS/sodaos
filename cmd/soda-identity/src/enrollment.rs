// Owner enrollment sessions, extracted from control.rs (A05.M).
use super::control::{new_id, zeroize, Controller, EnrollmentEntry, State};
use crate::wire::{Enrollment, Error, READY};

impl Controller {
    fn pending_enrollment(state: &State, owner: i64, provider_id: &str) -> bool {
        state.enrollments.values().any(|entry| {
            entry.owner == owner
                && entry.provider_id == provider_id
                && entry.result.connection.is_none()
                && entry.session.as_ref().is_some_and(|session| {
                    matches!(session.snapshot().state.as_str(), "pending" | "completed")
                })
        })
    }

    pub fn start_enrollment(
        &self,
        owner: i64,
        provider_id: &str,
        label: &str,
    ) -> Result<Enrollment, Error> {
        let mut state = self.lock();
        if !state.providers.contains_key(provider_id)
            || owner <= 0
            || label.len() > 100
            || label.is_empty()
        {
            return Err(Error::denied("identity authority denied"));
        }
        if Self::pending_enrollment(&state, owner, provider_id) {
            return Err(Error::busy());
        }
        let session = state.providers[provider_id].start(owner)?;
        let mut result = session.snapshot();
        result.provider_id = provider_id.to_string();
        if result.id.is_empty() {
            let _ = session.close();
            return Err(Error::uncertain());
        }
        state.enrollments.insert(
            result.id.clone(),
            EnrollmentEntry {
                owner,
                label: label.to_string(),
                provider_id: provider_id.to_string(),
                session: Some(session),
                result: result.clone(),
            },
        );
        Ok(result)
    }

    pub fn enrollment(&self, owner: i64, id: &str) -> Result<Enrollment, Error> {
        let mut guard = self.lock();
        let state: &mut State = &mut guard;
        let entry = state.enrollments.get_mut(id);
        let Some(entry) = entry else {
            return Err(Error::denied("identity authority denied"));
        };
        if entry.owner != owner {
            return Err(Error::denied("identity authority denied"));
        }
        let Some(session) = entry.session.as_ref() else {
            return Ok(entry.result.clone());
        };
        let mut result = session.snapshot();
        result.provider_id.clone_from(&entry.provider_id);
        if result.state != "completed" {
            entry.result = result.clone();
            return Ok(result);
        }
        let (mut connection, mut data) = match session.finish() {
            Ok(finished) => finished,
            Err(_) => {
                result.state = "failed".to_string();
                result.error = "Provider enrollment could not be retained".to_string();
                let _ = entry.session.as_ref().map(|s| s.close());
                entry.session = None;
                entry.result = result.clone();
                return Err(Error::internal("Provider enrollment could not be retained"));
            }
        };
        connection.provider_id.clone_from(&entry.provider_id);
        connection.id = new_id();
        connection.owner_id = owner;
        connection.label.clone_from(&entry.label);
        connection.generation = 1;
        connection.state = READY.to_string();
        let save = state.store.save_connection(&connection, &data);
        zeroize(&mut data);
        let _ = entry.session.as_ref().map(|s| s.close());
        entry.session = None;
        if let Err(err) = save {
            result.state = "failed".to_string();
            result.error = "Provider enrollment could not be retained".to_string();
            entry.result = result.clone();
            return Err(err);
        }
        result.connection = Some(connection);
        result.user_code.clear();
        result.verification_url.clear();
        entry.result = result.clone();
        Ok(result)
    }

    pub fn cancel_enrollment(&self, owner: i64, id: &str) -> Result<(), Error> {
        let mut state = self.lock();
        let Some(entry) = state.enrollments.get_mut(id) else {
            return Err(Error::denied("identity authority denied"));
        };
        if entry.owner != owner {
            return Err(Error::denied("identity authority denied"));
        }
        if let Some(session) = entry.session.as_ref() {
            session.close()?;
            entry.session = None;
        }
        entry.result.state = "canceled".to_string();
        entry.result.user_code.clear();
        Ok(())
    }
}
