#!/bin/bash

# CodeAtlas Test Coverage Script
# Generates comprehensive test coverage reports with detailed statistics

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
COVERAGE_FILE="coverage.out"
COVERAGE_HTML="coverage.html"
COVERAGE_THRESHOLD=80  # Minimum coverage percentage

# Function to print colored output
print_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Function to run tests with coverage
run_coverage() {
    print_info "Running tests with coverage..."
    go test $(go list ./... | grep -v /scripts | grep -v /test-repo) -coverprofile=$COVERAGE_FILE -covermode=atomic -v
    
    if [ $? -ne 0 ]; then
        print_error "Tests failed!"
        exit 1
    fi
    
    print_success "Tests completed successfully"
}

# Function to generate HTML report
generate_html() {
    print_info "Generating HTML coverage report..."
    go tool cover -html=$COVERAGE_FILE -o $COVERAGE_HTML
    print_success "HTML report generated: $COVERAGE_HTML"
}

# Function to show coverage statistics
show_stats() {
    print_info "Coverage Statistics:"
    echo ""
    
    # Overall coverage
    total_coverage=$(go tool cover -func=$COVERAGE_FILE | grep total | awk '{print $3}' | sed 's/%//')
    
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo -e "  ${BLUE}Total Coverage:${NC} ${GREEN}${total_coverage}%${NC}"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo ""
    
    # Check threshold
    if (( $(echo "$total_coverage < $COVERAGE_THRESHOLD" | bc -l) )); then
        print_warning "Coverage is below threshold (${COVERAGE_THRESHOLD}%)"
    else
        print_success "Coverage meets threshold (${COVERAGE_THRESHOLD}%)"
    fi
    
    echo ""
    print_info "Package Coverage:"
    echo ""
    
    # Package-level coverage
    go tool cover -func=$COVERAGE_FILE | grep -v "total:" | awk '{
        package = $1
        coverage = $3
        gsub(/%/, "", coverage)
        
        # Color based on coverage
        if (coverage >= 80) color = "\033[0;32m"  # Green
        else if (coverage >= 60) color = "\033[1;33m"  # Yellow
        else color = "\033[0;31m"  # Red
        
        printf "  %-60s %s%6s%%\033[0m\n", package, color, coverage
    }' | sort -t: -k1,1 -k2,2n
    
    echo ""
}

# Function to show uncovered code
show_uncovered() {
    print_info "Files with low coverage (<60%):"
    echo ""
    
    go tool cover -func=$COVERAGE_FILE | grep -v "total:" | awk '{
        package = $1
        coverage = $3
        gsub(/%/, "", coverage)
        
        if (coverage < 60) {
            printf "  %-60s %6s%%\n", package, coverage
        }
    }' | sort -t: -k1,1 -k2,2n
    
    echo ""
}

# Function to generate package summary
package_summary() {
    print_info "Package Summary:"
    echo ""
    
    # Extract unique packages and calculate average coverage
    go tool cover -func=$COVERAGE_FILE | grep -v "total:" | awk -F: '{print $1}' | sort -u | while read package; do
        if [ -n "$package" ]; then
            avg_coverage=$(go tool cover -func=$COVERAGE_FILE | grep "^$package:" | awk '{sum+=$3; count++} END {if(count>0) print sum/count; else print 0}' | sed 's/%//')
            
            # Color based on coverage
            if (( $(echo "$avg_coverage >= 80" | bc -l) )); then
                color="${GREEN}"
            elif (( $(echo "$avg_coverage >= 60" | bc -l) )); then
                color="${YELLOW}"
            else
                color="${RED}"
            fi
            
            printf "  %-50s ${color}%6.2f%%${NC}\n" "$package" "$avg_coverage"
        fi
    done
    
    echo ""
}

# Main execution
main() {
    echo ""
    echo "╔════════════════════════════════════════════════════════════╗"
    echo "║         CodeAtlas Test Coverage Report                    ║"
    echo "╚════════════════════════════════════════════════════════════╝"
    echo ""
    
    # Parse command line arguments
    case "${1:-all}" in
        "run")
            run_coverage
            ;;
        "html")
            if [ ! -f $COVERAGE_FILE ]; then
                print_error "Coverage file not found. Run 'make test-coverage' first."
                exit 1
            fi
            generate_html
            ;;
        "stats")
            if [ ! -f $COVERAGE_FILE ]; then
                print_error "Coverage file not found. Run 'make test-coverage' first."
                exit 1
            fi
            show_stats
            ;;
        "uncovered")
            if [ ! -f $COVERAGE_FILE ]; then
                print_error "Coverage file not found. Run 'make test-coverage' first."
                exit 1
            fi
            show_uncovered
            ;;
        "summary")
            if [ ! -f $COVERAGE_FILE ]; then
                print_error "Coverage file not found. Run 'make test-coverage' first."
                exit 1
            fi
            package_summary
            ;;
        "all")
            run_coverage
            show_stats
            show_uncovered
            package_summary
            generate_html
            ;;
        *)
            echo "Usage: $0 {run|html|stats|uncovered|summary|all}"
            echo ""
            echo "Commands:"
            echo "  run       - Run tests with coverage"
            echo "  html      - Generate HTML report"
            echo "  stats     - Show coverage statistics"
            echo "  uncovered - Show files with low coverage"
            echo "  summary   - Show package-level summary"
            echo "  all       - Run all of the above (default)"
            exit 1
            ;;
    esac
    
    echo ""
    print_success "Coverage analysis complete!"
    echo ""
}

main "$@"
