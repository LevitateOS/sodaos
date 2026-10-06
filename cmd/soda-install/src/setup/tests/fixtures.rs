use super::super::*;

pub(super) fn null_console() -> Console {
    Console::from_file("/dev/null", std::fs::File::open("/dev/null").unwrap())
}
