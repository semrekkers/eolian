function! EolianPatch()
    call s:signal("USR1")
endfunction

function! EolianBuild()
    call s:signal("USR2")
endfunction

function! s:signal(which)
    let pid = system("pgrep eolian")
    if pid == ""
        echom "Eolian isn't running"
        return
    endif
    echom system("kill -s " . a:which . " " . pid)
endfunction

command! EolianPatch call EolianPatch()
command! EolianBuild call EolianBuild()
