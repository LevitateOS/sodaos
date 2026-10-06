use super::super::*;
use super::fixtures::{
    ai_element, cert_from_tbs_parts, concat, err_text, gentime, null, octet, seq, std_parts,
    utctime, OID_SHA256_RSA,
};

fn validity_time(nb: &[u8], na: &[u8]) -> (i64, i64) {
    let mut parts = std_parts();
    parts[3] = seq(&concat(&[nb.to_vec(), na.to_vec()]));
    let cert = parse_certificate(&cert_from_tbs_parts(
        &parts,
        &ai_element(OID_SHA256_RSA, Some(&null())),
        &[1],
    ))
    .expect("validity parses");
    (cert.not_before, cert.not_after)
}

#[test]
fn validity_time_values() {
    // UTCTime year mapping and epoch.
    assert_eq!(
        validity_time(&utctime("700101000000Z"), &utctime("700102000000Z")),
        (0, 86400)
    );
    assert_eq!(
        validity_time(&utctime("500101000000Z"), &utctime("700101000000Z")).0,
        -631152000
    );
    assert_eq!(
        validity_time(&utctime("490101000000Z"), &utctime("700101000000Z")).0,
        2493072000
    );
    // GeneralizedTime.
    assert_eq!(
        validity_time(&gentime("19700101000000Z"), &utctime("700101000000Z")).0,
        0
    );
    // Numeric zones shift to UTC.
    assert_eq!(
        validity_time(&utctime("700101000000+0130"), &utctime("700101000000Z")).0,
        -5400
    );
    assert_eq!(
        validity_time(&utctime("700101030000-0130"), &utctime("700101000000Z")).0,
        16200
    );
    // Leap day.
    assert_eq!(
        validity_time(&utctime("000229120000Z"), &utctime("700101000000Z")).0,
        951825600
    );
}

#[test]
fn validity_time_rejections() {
    // Month 13.
    let mut parts = std_parts();
    parts[3] = seq(&concat(&[
        utctime("701301000000Z"),
        utctime("700102000000Z"),
    ]));
    assert_eq!(
        err_text(parse_certificate(&cert_from_tbs_parts(
            &parts,
            &ai_element(OID_SHA256_RSA, Some(&null())),
            &[1]
        ))),
        "x509: malformed UTCTime"
    );
    // Hour 24.
    parts[3] = seq(&concat(&[
        utctime("700101240000Z"),
        utctime("700102000000Z"),
    ]));
    assert_eq!(
        err_text(parse_certificate(&cert_from_tbs_parts(
            &parts,
            &ai_element(OID_SHA256_RSA, Some(&null())),
            &[1]
        ))),
        "x509: malformed UTCTime"
    );
    // GeneralizedTime wrong length.
    parts[3] = seq(&concat(&[
        gentime("1970010100000Z"),
        utctime("700102000000Z"),
    ]));
    assert_eq!(
        err_text(parse_certificate(&cert_from_tbs_parts(
            &parts,
            &ai_element(OID_SHA256_RSA, Some(&null())),
            &[1]
        ))),
        "x509: malformed GeneralizedTime"
    );
    // Non-UTCTime/GeneralizedTime tag.
    parts[3] = seq(&concat(&[
        octet(b"700101000000Z"),
        utctime("700102000000Z"),
    ]));
    assert_eq!(
        err_text(parse_certificate(&cert_from_tbs_parts(
            &parts,
            &ai_element(OID_SHA256_RSA, Some(&null())),
            &[1]
        ))),
        "x509: unsupported time format"
    );
}
