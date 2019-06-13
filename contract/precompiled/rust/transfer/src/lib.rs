#![no_std]

use core::alloc::{GlobalAlloc, Layout};

extern crate wee_alloc;

#[global_allocator]
static ALLOC: wee_alloc::WeeAlloc = wee_alloc::WeeAlloc::INIT;

#[panic_handler]
pub fn panic(_info: &core::panic::PanicInfo) -> ! {
    loop {}
}

extern "C" {
    fn sm_vm_get_sender_addr() -> i32;
// fn sm_vm_get_balance(addr_ptr: i32, balance_ptr: i32, balance_len: i32) -> i32;
// fn sm_vm_set_balance(addr_ptr: i32, balance_ptr: i32, balance_len: i32);
}

#[allow(non_snake_case)]
#[inline(never)]
#[no_mangle]
pub extern "C" fn Allocate(bytes_count: i32) -> i32 {
    let layout = Layout::from_size_align(bytes_count as usize, 4).unwrap();

    let ptr = unsafe { ALLOC.alloc(layout) };

    ptr as i32
}

#[allow(non_snake_case)]
#[no_mangle]
pub extern "C" fn Transfer(to_addr_ptr: i32, amount_ptr: i32, amount_len: i32) -> i32 {
    // allocate memory for the balance reading
    // 32 bytes should be enough
    // sm_alloc(32)

    // let balance_ptr: i32 = 0;
    // let balance_len: i32 = 32;
    // let from_addr = unsafe { sm_vm_get_sender_addr() };

    // let from_balance = unsafe { sm_vm_get_balance(from_addr, 20, balance_ptr, balance_len) };
    // let to_balance = unsafe { sm_vm_get_balance(to_addr, addr_len) };

    // asserting we won't have underflow
    // if from_balance <= amount {
    //     return -1;
    // }
    //
    // // asserting we won't have overflow
    // if to_balance + amount < to_balance {
    //     return -1;
    // }
    //
    // let from_new_balance = from_balance - amount;
    // let to_new_balance = to_balance + amount;
    //
    // unsafe {
    //     sm_vm_set_balance(from_addr, addr_len, from_new_balance);
    //     sm_vm_set_balance(to_addr, addr_len, to_new_balance);
    // }

    return 11;
}
