require "fileutils"

out_dir = "../out"

oldGoPath = ENV["GOPATH"]
ENV["GOPATH"] = Dir.pwd

at_exit {
	ENV["GOPATH"] = oldGoPath
}

def exec_or_die(cmd, current_dir = nil)
	orgDir = Dir.pwd
	Dir.chdir(current_dir) if current_dir != nil
	
	res = `#{cmd}`
	
	Dir.chdir(orgDir)

	if $?.exitstatus != 0
		print(res)
		exit(false)
	end
end

case ARGV[0]
	when "prepare"	then
		exec_or_die("go install github.com/IMQS/messaging/cmd")
		FileUtils.cp("bin/cmd.exe", out_dir + '/bin/imqsmessaging.exe')
	when "test_unit" then
	when "test_integration" then
end