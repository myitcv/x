# We can put whatever we like here. The START delimeter below marks the actual
# start of our script
echo This will not be seen

# **START**

comment "We can use the comment function to output comments.
This is only really useful when running with -out std"
comment
comment "assert when an exit code is something non-zero (else the script will fail)"
comment

# block: assert
false
assert "$? -eq 1" $LINENO

comment
comment "catfile's output is most useful with PrintBlockOut in mdreplace"
comment

# block: catfile
cat <<EOD > a_file.txt
Hello, world
EOD
catfile a_file.txt

# egrunner_envsubst: +repo
# egrunner_replace: "^Good morning, (.*)$" "Hi $1!"

comment
comment 'In this script we have defined the following directives:

egrunner_envsubst: +repo
egrunner_replace: "^Good morning, (.*)$" "Hi $1!"

Hence we get the following:'
comment

# block: directives
export repo=X
echo $repo
echo "Good morning, Rob"

