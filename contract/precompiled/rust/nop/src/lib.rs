#![no_std]

#[panic_handler]
pub fn panic(_info: &core::panic::PanicInfo) -> ! {
    loop {}
}

#[allow(non_snake_case)]
#[no_mangle]
fn Execute() {
    let _ = 10;
}
